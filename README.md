# Zipper Web server (for the lack of a better name)

## Purpose

I wanted a tool that allows me to share a specific directory online. Users will see a simple directory listing that allows them to download individual files or multiple files and folders as a ZIP file.
A nice feature of this tool, is that, it creates the ZIP file on the fly while it is being downloaded, therefore not occupying any disk space for temporary files.

Written in Go, Zipper runs on Windows, Linux, Raspberry Pi and many more operating systems and CPU architectures.

## Usage

Specify a directory to be shared with the -root flag or simply place the executable in the directory you would like share and run it.