#include <sys/wait.h>
#include <errno.h>
#include <unistd.h>
#include <stdio.h>
#include <ctime>
#include <cstdlib>
#include <cstdarg>
#include <cstring>
#include <vector>

#include "ExecDemo.h"

#define MAX_PIPE_BUFFER 1024

extern char** environ;

int kill_r(pid_t pgid) {
    if (pgid <= 0) {
        return -1;
    }

    pid_t pid;
    if ((pid = fork()) < 0) {
        return -101;
    } else if (pid == 0) {
        if (setuid(0) < 0) { // Revert to root
            //Ignore
        }
        char cmdString[256];
        memset(cmdString, 0, 256);
        // Need to kill process group
        sprintf(cmdString, "kill -9 -%d", pgid);
        execle("/bin/sh", "sh", "-c", cmdString, (char *)0, environ);
        //Never go here
        exit(127);
    } else {
        int status = -1;
        pid_t childPid;
        do {
           childPid = waitpid(pid, &status, 0);
        } while (childPid < 0 && errno == EINTR);

        if (WIFEXITED(status) && (WEXITSTATUS(status) == 0)) {
            return 0;
        } else {
            return -1;
        }
    }
}

void ConvertStringToList(std::vector<std::string> &stringsVector, const std::string &strToBreak) {
    static const char * whitespaces = " \t\n";
    stringsVector.clear();

    if ( strToBreak.length() == 0 ) {
        return;
    }

    size_t curpos = strToBreak.find_first_not_of(whitespaces);
    size_t pos;
    while(curpos != std::string::npos) {
        char charToMatch = strToBreak.at(curpos);
        if( '\'' == charToMatch || '\"' == charToMatch ) {
            pos = strToBreak.find( charToMatch, curpos + 1 );
            if ( pos == std::string::npos ) {
                pos = strToBreak.find_first_of( whitespaces, curpos + 1 );
            } else {
                ++pos;
            }
        } else  {
            // no matching needed.
            pos = strToBreak.find_first_of( " \t\n\'\"", curpos + 1 );
        }
        if ( std::string::npos == pos ) {
            stringsVector.push_back( strToBreak.substr(curpos) );
        } else {
            stringsVector.push_back( strToBreak.substr(curpos, pos - curpos) );
        }
        curpos = strToBreak.find_first_not_of(whitespaces, pos);
    }
}

char** ListToArray(const std::vector<std::string> &listOfStrings ) {
    char** ret = new char*[listOfStrings.size() + 1];
    for (size_t i = 0; i < listOfStrings.size(); i++) {
        size_t numChars = listOfStrings[i].length();
        char* arg = new char[numChars + 1];
        strncpy(arg, listOfStrings[i].c_str(), numChars);
        arg[numChars] = '\0';
        ret[i] = arg;
    }
    ret[listOfStrings.size()] = NULL;
    return ret;
}

int runCommand(const char* command, std::string & strOutput, int timeout, bool needBreakDown, ...) {
    std::vector<std::string> argumentList;
    argumentList.push_back(command);

    if (needBreakDown) {
        std::string strCommand(command);
        ConvertStringToList(argumentList, strCommand);
        for (int i = 0 ; i < argumentList.size() ; i++) {
            printf("Debug:: %s\n", argumentList[i].c_str());
        }
    } else {
        va_list args;
        va_start(args, false);
        const char* arg = va_arg(args, const char*);
        while (arg != NULL) {
            argumentList.push_back(std::string(arg));
            arg = va_arg(args, const char*);
        }
        va_end(args);
    }

    int fd[2];
    if (pipe(fd) < 0) {
        return -102;
    }

    pid_t pid;
    if ((pid = fork()) < 0) {
        return -101;
    } else if (pid == 0) {
        //Child process
        setsid(); //for kill
        close(fd[0]); //close read end
        //assign the stdout/stderr to write end
        if (fd[1] != fileno(stdout)) {
            dup2(fd[1], fileno(stdout));
        }
        if (fd[1] != fileno(stderr)) {
            dup2(fd[1], fileno(stderr));
        }
        execve(argumentList.front().c_str(), ListToArray(argumentList), environ);
        //Should never go here
        printf("Should never to see this\n");
        exit(127);
    } else {
        //Parent process
        close(fd[1]); //close write end
        int rc = -1;
        long timeLeft = timeout;
        time_t lastTime, now;
        time(&lastTime);
        while (1) {
            time(&now);
            timeLeft -= (now - lastTime);
            lastTime = now;
            if ( timeLeft < 0 ) {
                timeLeft = 0;
            }
            struct timeval waitperiod = {timeLeft, 0};
            fd_set stSocket;
            FD_ZERO(&stSocket);
            FD_SET(fd[0], &stSocket);
            // A null waitperiod means wait indefinitely.
            int nReady = select (fd[0] + 1, &stSocket, NULL, NULL, (timeout > 0) ? &waitperiod : NULL);
            if (nReady < 0) { // An error occured.
                break;
            }
            if (nReady == 0) { // Timeout occured.
                rc = -103;
                break;
            }
            if (FD_ISSET(fd[0], &stSocket)) {
                // Buffer from stdout/stderr
                char outString[MAX_PIPE_BUFFER];
                int nRead;
                if ((nRead = read(fd[0], outString, MAX_PIPE_BUFFER - 1)) <= 0) {
                    break;
                }
                outString[nRead] = '\0';
                strOutput += outString;
            }
        }
        close(fd[0]); //read end
        kill_r(pid); // kill the child process (if it is not already dead)
        // If it was not because of timeout, collect the child process' status to prevent zombie processes
        // Need to initialize status to -1, otherwise Solaris 10 returns true for WIFEXITED and will cause
        // rc to be set incorrectly.
        int status = -1;
        if (rc != -103) {
            pid_t childPid;
            do {
                childPid = waitpid(pid, &status, 0);
            } while (childPid < 0 && errno == EINTR);
        }
        // Get the actual exit code set by the process.
        if (WIFEXITED(status)) {
            rc = WEXITSTATUS(status);
        }
        return rc;
    }
}
