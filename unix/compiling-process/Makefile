hello.i: hello.c
	gcc -E hello.c -o hello.i

hello.s: hello.i
	gcc -S hello.i -o hello.s

hello.o: hello.s
	gcc -c hello.s -o hello.o

hello: hello.o
	gcc hello.o -o hello

all: hello

.PHONY: clean
clean:
	 rm -f hello.i hello.s hello.o hello