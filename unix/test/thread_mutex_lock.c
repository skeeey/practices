#include <pthread.h>
#include <stdlib.h>
#include <unistd.h>
#include <stdio.h>

int gcount = 0;
pthread_mutex_t mymutex = PTHREAD_MUTEX_INITIALIZER;

void* run (void* arg) {
    int i;
    for (i = 0; i < 20; i++) {
        pthread_mutex_lock(&mymutex);
        //gcount = gcount + 1;
        int tmp = gcount;
        tmp = tmp + 1;
        printf(".");
        fflush(stdout);
        sleep(1);
        gcount = tmp;
        pthread_mutex_unlock(&mymutex);
    }
    return NULL;
}

int main (void) {
    pthread_t newThread;
    if (pthread_create(&newThread, NULL, run, NULL) != 0) {
        printf("error creating thread.");
        abort();
    }

    int i;
    for (i = 0; i < 20; i++) {
        pthread_mutex_lock(&mymutex);
        gcount = gcount + 1;
        pthread_mutex_unlock(&mymutex);
        printf("-");
        fflush(stdout);
        sleep(1);
    }

    if (pthread_join(newThread, NULL)) {
        printf("error joining thread.");
        abort();
    }
    printf("\nglobal count = %d\n", gcount);
    exit(0);
}
