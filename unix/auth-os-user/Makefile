auth_os_user.o: auth_os_user.c
	gcc -c auth_os_user.c

auth_os_user: auth_os_user.o
	gcc -o auth_os_user auth_os_user.o -lcrypt

all: auth_os_user

.PHONY: clean
clean:
	 rm -f *.o auth_os_user
