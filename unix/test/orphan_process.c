#include <stdio.h>
#include <unistd.h>

int main() {
    pid_t fpid = fork();
    if (fpid < 0) {
        printf("Faild to fork process\n");
        return 1;
    }

    if (fpid == 0) {
        printf("Child Process: %d, %d\n", getpid(), getppid());
        sleep(30);
        //This process will be killed by init process, because
        //its parent process does not wait for it
        printf("End Child Process: %d, %d\n", getpid(), getppid());
    } else {
        printf("Parent Process: %d\n", getpid());
        sleep(5);
        printf("End Parent Process: %d\n", getpid());
    }
    return 0;
}
