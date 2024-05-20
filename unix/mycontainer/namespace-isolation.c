#define _GNU_SOURCE
#include <sys/mount.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <stdio.h>
#include <sched.h>
#include <signal.h>
#include <unistd.h>

#define STACK_SIZE (1024*1024)

static char child_stack[STACK_SIZE];

char* const child_args[] = {
    "/bin/bash",
    NULL
};

int pipefd[2]; //pipefd[0] -> read, pipefd[1] -> write

void set_uid(pid_t pid, int inside_id, int outside_id) {
    char path[256];
    sprintf(path, "/proc/%d/uid_map", pid);
    FILE* uid_map = fopen(path, "w");
    if (NULL == uid_map) {
        perror(path);
        return;
    }
    fprintf(uid_map,"%d %d %d", inside_id, outside_id, 1);
    fclose(uid_map);
}

void set_gid(pid_t pid, int inside_id, int outside_id) {
    char path[256];
    sprintf(path, "/proc/%d/gid_map", pid);
    FILE* gid_map = fopen(path, "w");
    if (NULL == gid_map) {
        perror(path);
        return;
    }
    fprintf(gid_map, "%d %d %d", inside_id, outside_id, 1);
    fclose(gid_map);
}

int container(void* args) {
    char ch;
    close(pipefd[1]);
    read(pipefd[0], &ch, 1); //wait parent process
    printf("[inside] Container Process ID %d, UID=%d, GID=%d\n", getpid(), getuid(), getgid());
    //set hostname
    sethostname("NewContainer", 13);
    // make sure mount propagation is private
    mount("none", "/", NULL, MS_REC | MS_PRIVATE, NULL);
    // create a new mount /proc
    mount("proc", "/proc", "proc", 0, NULL);
    execv(child_args[0], child_args);
    return 1;
}

int main() {
    pipe(pipefd);
    printf("Container Parent Process %d\n", getpid());
    printf("%d, %d\n", MS_BIND, MS_SHARED);
    int child_pid = clone(container,
        child_stack + STACK_SIZE, //The location of the stack used by child process, from child_stack to STACK_SIZE
        //CLONE_NEWUTS | SIGCHLD,
        //CLONE_NEWPID | CLONE_NEWIPC | CLONE_NEWUTS | SIGCHLD,
        CLONE_NEWNS | CLONE_NEWPID | CLONE_NEWIPC | CLONE_NEWUTS | SIGCHLD,
        //CLONE_NEWUSER cannot be supported by 3.10.0-514.21.1.el7.x86_64, may be supported by 4.0+
        //CLONE_NEWUSER | CLONE_NEWNS | CLONE_NEWPID | CLONE_NEWIPC | CLONE_NEWUTS | SIGCHLD,
        NULL);
    if (-1 == child_pid) {
        perror("Failed to create child process");
        return 1;
    }
    printf("[outside] Container Process ID %d, UID=%d, GID=%d\n", child_pid, getuid(), getgid());
    //set_uid(child_pid, 0, getuid());
    //set_gid(child_pid, 0, getgid());
    sleep(10);
    close(pipefd[1]); // send a singal to notify child can start
    waitpid(child_pid, NULL, 0);
    printf("Container Parent Process End\n");
    return 0;
}
