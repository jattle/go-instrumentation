package main

// go-instrument-tool accept one go source file and do auto patch code instrumentation, it doesnt know
// which source file to instrument. for one golang project, most of times we are only intrested in itself,
// sometimes we alse want to diagnose logic of its dependent modules, so we need this helper tool.
// it can do followings:
// 1. download project dep modules;
// 2. select modules you are intrested, which matched path pattern;
// 3. copy module source to pkgModDir;
// 4. generate replaces for these dep modules and append them into project gomod file.