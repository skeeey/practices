#include <stdio.h>

struct str {
    int len;
    char s[0];
};
 
struct foo {
    struct str *a;
};
 
int main(void) {
    struct foo f = {0};
    //try to get the address of s
    if (f.a->s) {
        printf("%x\n", f.a->s);
        // printf(f.a->s); //crash, try to get value of a
    }
    return 0;
}
