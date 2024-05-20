#include <stdio.h>

#include "ExecDemo.h"

int main(void) {
    std::string output;
    std::string command = "/usr/bin/java";
    std::string arg0 = "-version";
    int ret = runCommand(command.c_str(), output, 5, false, arg0.c_str(), (char*) 0);
    printf("exit code: %d\n", ret);
    printf("%s\n", output.c_str());
}
