# unix-practice
Unix programming practice

## auth-os-user

Authenticate OS user
- `255` - password is not match
- `254` - user does not exist

## compiling-process

Compiling process demo

Some ELF (Executable and Linking Format) file analyzing commands

```
file        # 探测给定文件的类型
ldd         # 打印程序或者库文件所依赖的共享库列表， --version 查看 glibc 的版本
ldcongfig   # 在默认搜寻目录 /lib 和 /usr/lib 以及动态库配置文件 /etc/ld.so.conf 内所列的目录下，
            # 搜索出可共享的动态链接库（格式如lib*.so*）,进而创建出动态装入程序(ld.so)所需的连接
            # 和缓存文件。缓存文件默认为/etc/ld.so.cache，此文件保存已排好序的动态链接库名字列
            # 表，为了让动态链接库为系统所共享，需运行动态链接库的管理命令ldconfig，此执行程序存放在/sbin目录下。
nm          # 查看 ELF 文件的符号表信息
readelf     #用来显示一个或者多个elf格式的目标文件的信息，可以通过它的选项来控制显示哪些信息
objdump     # 查看目标文件或者可执行的目标文件的构成
strip       # 通过除去绑定程序和符号调试程序使用的信息，减少扩展公共对象文件格式（XCOFF）的对象文件的大小。
```

## exec-cmd

Run `/usr/bin/java -version` command

## mycontainer

Linux namespace isolation demo

## test

Some interesting cases

```
gcc thread_mutex_lock.c -lpthread
```
