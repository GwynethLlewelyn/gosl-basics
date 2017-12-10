# gosl-basics

A basic example of how to develop external web services for Second Life/OpenSimulator using the Go programming language.

## Why use Go?

This is a basic example (although from the code it might seem complex!) on how to interface an in-world object in Second Life/OpenSimulator with a web application compiled with Go. Two huge advantages that Go has over more common web-based solutions such as PHP, .NET, Perl, Python, Java, Node.js, etc. is that unlike any of those _there is no web server configuration necessary_. Go has its own built-in web server, and Google designed it for efficiency. What this means is that you just need to compile the application and run it — no configuration necessary; no dependencies; no worrying if you have the 'right' version of the language, or the 'right' versions of the libraries and DLLs needed to run it... there is nothing of the sort! You just run the application written in Go and that's it. The second huge advantage is that Go is a _compiled language designed to be used for system administrators_ (and not necessarily a 'cutey' language for Web designers, no offence meant!). There are no 'virtual machines', no 'interpreters', no 'just-in-time compilers', no 'opcaches', or whatever is currently fashionable to offer in a modern programming language. Go compiles to efficient machine code, end of story: it can't get any faster than that (unless you write your own web server in machine code!). The trade-off? Well, you can't simply grab the application compiled in a Windows machine and expect it to run on a Linux box; you have to recompile it again. That's what _allegedly_ is the advantage of languages such as Java — write once, run on every platform that supports Java. In theory. In practice, well... no comments :)

Because Go effectively comes with its own webserver... the code might be a little more lengthy than what you'd expect for a simple thing like this. But I've been mean and added a lot of bells and whistles, so that you don't get merely a 'skeleton' project with no clue how to extend it. `gosl-basics` may be basic, but it's a fully-fledged web server, able to run as either a stand-alone server or as a FastCGI application, including an interactive shell (or command line) to do some quick testing, it parses command-line arguments, and has a fairly sophisticated logging system, which will do automatic rotation and so forth... all the tools that you need to build a 'serious' application, in fact. In case you are wondering: no, I didn't develop any of those features, they are all shared by Go programmers via open-source repositories, which extend the 'core' functionality of Go (meaning the libraries provided by the original developers and maintainers) in a very easy way.

## Yes, fine, but what does this application actually _do_?

I could have provided a very basic 'echo' service, which is what 99.9% of people do when demonstrating new features in a web service; but I found out from experience that such an 'echo' service almost always uses nasty tricks, because it's so simple, to make the code minimalistic. And then it's actually very hard to figure out how to use it in a 'real world' situation!

Instead, I've made another version of the (in)famous **name2key** database. If you're a LSL scripter, you're aware that Linden Lab provides a way to get the name of an avatar (or any object, really) by supplying its UUID (or key), but the reverse is not true. Why? Because it could be abused — if you have the avatar keys for _all_ the avatars, then you could theoretically spam them all with instant messages, and there would be no way for them to 'unlist' themselves from such a spam list. So Linden Lab deliberately never provided that functionality. The only alternative is to patiently grab names & keys from avatars who pass by a sensor, touch an object, sit on an object, and so forth, and accumulate those names and keys on an external database (which will have literally millions of entries). Several people have done such databases in the past, and I still have one (running under PHP) which also served as a purpose to illustrate the ability that Second Life has to store persistent information in external servers. That was more than a decade ago!

The most famous — and most complete! — name2key database had been compiled by the (in)famous [W-Hat group](http://w-hat.com/). I'll leave as an exercise for you to look them up and their history. They have kindly left their service behind, and even given away their source code, placing it in the public domain. There is a catch, though: they fancy an obscure programming language called 'Clojure', which is basically LISP compiled to a Java Virtual Machine. LISP is the kind of language that only ubergeeks know; developed in 1958 (yes, seriously!), it was the first programming language for artificial intelligence. It still has lots of uses, even outside the field of AI (the famous computer-aided design platform AutoCAD used a dialect of LISP for ages to extend its functionality), but it's something that most people will never encounter in their lives :-) Now I have nothing against LISP (it used to be one of my favourite languages), but I hate the idea that I need to install a whole Java environment, plus configuring Clojure properly, just to be able to use their (simple) code.

Instead, I rather prefer to show how it's done in Go.

## Installation overview

Requirements:
- You need to have [Go](https://golang.org) installed and configured properly
- Your computer/server needs to have a publicly accessible IP (and set it up with a **dynamic DNS provider** such as [no-ip.com](https://www.noip.com/remote-access)); you _can_ run it from home if you wish

Finally, `go get git.gwynethllewelyn.net/GwynethLlewelyn/gosl-basics.git`, and you _ought_ to have a binary executable file in `~/go/bin` called `gosl-basics.git`. Just run it!

Then grab the two LSL scripts, `query.lsl` and `touch.lsl`, which ought to be in `~go/src/git.gwynethllewelyn.net/GwynethLlewelyn/gosl-basics`. The first runs queries: you touch to activate it, write the name of an avatar in chat using '/5 firstname lastname', and it replies with the avatar key (after a timeout, the script resets, giving another person a chance to try it out). The second script is to be placed on anything touchable to grab avatar names and keys and send it to your own database. You can, of course, use other methods to grab those; as an exercise, use a Sensor instead, or — even better and consuming much less resources — use a transparent, phantom prim across a place where people have no choice but to go through, and register names & keys as avatars 'bump' into that prim! There are more exotic alternatives, such as registering the name & key when an avatar sits on the prim; or using a llCastRay to figure out if there is anybody nearby. Lots of possibilities! :-) 

If something went wrong, you might need to download all external packages manually. Theoretically, the `go get` command is supposed to be clever enough to figure out everything it needs and download _everything_ automatically, but we all know how complex systems manage to fail, don't we? So, if needed, these are all the external packages (i.e. not part of the standard library) to be downloaded:

	go get	"github.com/spf13/pflag"
	go get	"github.com/dgraph-io/badger"
	go get	"github.com/dgraph-io/badger/options"
	go get	"github.com/fsnotify/fsnotify"
	go get	"github.com/op/go-logging"
	go get	"github.com/spf13/viper"
	go get	"github.com/syndtr/goleveldb/leveldb"
	go get	"github.com/syndtr/goleveldb/leveldb/util"
	go get	"github.com/tidwall/buntdb"
	go get	"gopkg.in/natefinch/lumberjack.v2"

These are for 3 different key/value databases; handling command-line flags and a configuration file; a powerful logging system; and a way to rotate logs (a lumberjack... chops down logs... get it? :) Aye, Gophers have a sense of humour).


## Configuration
You can run the executable either as:

      --database string   Database type (badger, buntdb, leveldb) (default "badger")
      --dir string        Directory where database files are stored (default "slkvdb")
      --import string     Import database from W-Hat (use the csv.bz2 version) (default "name2key.csv.bz2")
      --nomemory          Attempt to use only disk to save memory on Badger (important for shared webservers)
      --port string       Server port (default "3000")
      --server            Run as server on port 3000
      --shell             Run as an interactive shell
      	 
Basically, if you are running your own server (possibly at home!), you only need to run `gosl-basics -server`. You don't need to set up Apache or nginx or any other third-party software; `gosl-basics` is a fully standalone application and does not depend on anything.

If you're using a shared web server, like the ones provided by [Dreamhost](https://dreamhost.com), then you will very likely want to run `gosl-basics` as a FastCGI application. Why? Well, Dreamhost's Terms of Service explicitly forbid any application to be run all the time (to conserve memory, CPU slices, and, well, open ports). Instead, they offer the ability to run applications as FastCGI applications instead (under their own Apache). This is actually a very cool interface (as opposed to the ancient, non-fast CGI...) allowing parts of the setup of the application to be done when it is called the first time, and then launch requests on demand. _If_ there is a _lot_ of traffic, the application will actually remain active in memory/CPU for a long time! If it only gets sporadic calls once in a while, well, in that case, the application gets removed from memory until someone calls the URL again. I have not tested exhaustively, and this will certainly depend from provider to provider, but Dreamhost seems to allow the application to remain active in memory and in the process space for 30-60 seconds.

Remember to set your URLs on each of the two LSL scripts!

For the standalone server: this will be something like `http://your.server.name:3000/`.

For FastCGI: If your base URL (i.e. pointing to the directory where you have installed `gosl-basics`) is something like `http://your.hosted-server.name/examples` then your URLs will be something like `http://your.hosted-server.name/examples/gosl-basics`. Some providers might force you to add an extension like `.fcgi`, meaning that you will have to remember to add that explicitly every time you compile the application again (or just use a [Makefile](https://www.devroom.io/2015/10/03/a-makefile-for-golang-cli-tools/) for that!).

Note that the first time ever the application runs, it will check if the database directory exists, and if not, it will attempt to create it (and panic if it cannot create it, due to permissions — basically, if it can't create a directory, it won't be able to create the database files either). You can define a different location for the database; this might be important when using FastCGI on a shared server, because you might wish to use a private area of your web server, so that it cannot be directly accessed.

The `--shell` switch is mostly meant for debugging, namely, to figure out if the database was loaded correctly (see below) and that you can query for avatar names and/or UUIDs to see if they're in the database. Remember to run from the same place where the database resides (or pass the appropriate `--dir`command). Also, you will get extra debugging messages.

The `--nomemory` switch may seem weird, but in some scenarios, like shared servers with FastCGI and using the Badger database, the actual memory consumption may be limited, so this attempts to reduce the amount of necessary memory (things will run much slower, though; the good news is that there is _some_ caching).

See below for instructions for importing CSV bzip2'ed databases using `--import`. The CSV file format is one pair **UUID,Avatar Name** per line, and all of that bzip2'ed. 

## Limitations

I found that somehow the standard FastCGI package in Go seems to be limited to just one handler on the path router, so I took that into account and simply tried to figure out what was requested based on the valid parameters (`name` and `key`) passed via GET or POST. A more complex real-world example might require a much more sophisticated router; possibly the [Gorilla](https://github.com/gorilla/mux) router would handle FastCGI better, I don't know.

Note that the current version can be used as a direct replacement for [W-Hat's name2key](http://w-hat.com/#name2key). There is now a 'compatibility mode' with W-Hat: if on the calling URL the extra parameter `"compat=false"` is passed, then cute messages are sent back; if not, then it just sends back the UUID (or the avatar name). Further compatibility with W-Hat's database is not built-in.

To actually _use_ the W-Hat database, you need to download it first and import it. This means using the `--import` command (use the `name2key.csv.bz2`version). W-Hat still updates that database daily, so, with some clever `cron` magic, you might be able to get a fresh copy every day to import. Note that the database is supposed to be unique by name (and the UUIDs are not supposed to change): that means that you can import the 'new' version over an 'old' version, and only the relevant entries will be changed. Also, if you happen to have captured new entries (not yet existing on W-Hat's database) then these will _not_ be overwritten (or deleted) with a new import. To delete an old database, just delete the directory it is in.

Importing the whole W-Hat database, which has a bit over 9 million entries, took on my Mac 1 minute and 38 seconds. Aye, that's quite a long time. On a shared server, it can be even longer. The code has been substantially changed to use `BatchSet` which is allegedly the recommended way of importing large databases, but even in the scenario to consume as little memory as possible, it will break most shared servers, simply because Go's garbage collector will not be fast enough to clean up after each batch is sent — I may have to take a look at how to do this better, perhaps with less concurrency.

This also works for OpenSimulator grids and you can use the same scripts and database if you wish. Currently, the database stores an extra field with UUID, which is usually set to the name of the grid. Linden Lab sets these as 'Production' and 'Testing' respectively; other grid operators may use other names. There is no guarantee that every grid operator has configured their database with an unique name. Also, because of the way the key/value database works, it _assumes_ that all avatar names are unique across _all_ grids, which may not be true: only UUIDs are guaranteed to be unique. The reason why this was implemented this way was that I wanted _very fast_ name2key searches, while I'm not worried about very slow key2name searches, since those are implemented in LSL anyway. To make sure that you can have many avatars with the same name but different UUIDs, well, it would require a different kind of database. Also, what should a function return for an avatar with three different UUIDs? You can see the problem there. Therefore I kept it simple, but be mindful of this limitation.

Oh, and with 9 million entries, this is slow.