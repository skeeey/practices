#include <unistd.h>  
#include <string.h>  
#include <stdlib.h>  
#include <stdio.h>  
#include <sys/wait.h>  

int pipefd[2]; //pipefd[0] -> read, pipefd[1] -> write

int main(void) {
    if (pipe(pipefd) == -1) {
        perror("Failed to open pipe");
        return 1;
    }

    pid_t pid = fork();
    if (pid < 0) {
        perror("Failed to fork process");
        return 1;
    } else if (pid == 0) {
        close(pipefd[1]); //close write, keep read
        char buf[1024];
        int len = read(pipefd[0], buf, sizeof(buf));
        write(STDOUT_FILENO, buf, len);
        close(pipefd[0]);
    } else {
        close(pipefd[0]); //close read, keep write
        const char *p = "test for pipe\n";
        write(pipefd[1], p, strlen(p));
        wait(NULL);
        close(pipefd[1]);
    }
    return 0;
}
