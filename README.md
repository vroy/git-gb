# gb

List branches with additional information similar to the GitHub branches view.

* the timestamp of the last revision
* branch name
* number of revisions behind master
* number of revisions ahead of master

The output is sorted in chronological order - your last modified branches appear at the bottom of your prompt so you don't have to scroll.

    ~/c/gb:master$ gb
    2014-11-22 20:54PM | foobar                   | behind:   15 | ahead:    2
    2014-11-24 21:18PM | readme                   | behind:    0 | ahead:    1


## Installation

The install script will do its best to install dependencies before compiling.

    curl -sSL https://raw.githubusercontent.com/exploid/gb/master/install | bash -s stable

Or alternatively, after making sure that [cmake](http://www.cmake.org/) is installed:

    git clone git@github.com:exploid/gb.git
    cd gb
    make deps
    make
    sudo make install
