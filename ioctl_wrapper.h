#include <sys/ioctl.h>

typedef struct winsize ttysize_t;
void myioctl(int i, unsigned long l, ttysize_t* t){ioctl(i,l,t);}
