ExecDemo.o: ExecDemo.cpp ExecDemo.h
	g++ -c ExecDemo.cpp

ExecDemoTest.o: ExecDemoTest.cpp ExecDemo.h
	g++ -c ExecDemoTest.cpp

exec_demo: ExecDemo.o ExecDemoTest.o
	g++ -o exec_demo ExecDemo.o ExecDemoTest.o

all: exec_demo
	
.PHONY: clean
clean:
	rm -f *.o test_thread exec_demo
