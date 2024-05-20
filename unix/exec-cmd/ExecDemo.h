#ifndef EXECDEMO_H
#define EXECDEMO_H
#endif

#include <string>

int runCommand(const char* command, std::string & strOutput, int timeout, bool needBreakDown, ...);
