#define _XOPEN_SOURCE

#include <stdio.h>
#include <string.h>
#include <shadow.h>
#include <unistd.h>

int authenticate(const char *user, const char *pwd) {
    struct spwd *sp;

    // Need root permission
    if (geteuid() != 0) {
        return 13;
    }

    // The getspnam() function returns a pointer to a structure containing
    // the broken-out fields of the record in the shadow password database
    // that matches the username name.
    // The user does not exist
    if ((sp = getspnam(user)) == (struct spwd*) 0) {
        return 254;
    }

    // The password is not right
    if(strcmp(crypt(pwd, sp->sp_pwdp), sp->sp_pwdp) != 0) {
        return 255;
    }

    return 0;
}

int main(int argc, char **argv) {
    if (argc != 3) {
        return 22;
    }
    return authenticate(argv[1], argv[2]);
}
