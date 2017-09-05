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

Taking that into account, all you need now to do is add the external packages used by this project:
```
go get github.com/dgraph-io/badger
go get github.com/op/go-logging
go get gopkg.in/natefinch/lumberjack.v2
```
which are, respectively, a very fast name/key database; a powerful logging system; and a way to rotate logs (a lumberjack... chops down logs... get it? :) Aye, Gophers have a sense of humour).

Finally, `go get git.gwynethllewelyn.net/GwynethLlewelyn/gosl-basics.git` and you _ought_ to have a binary executable file in `~/go/bin` called `gosl-basics`. Just run it!

Then grab the two LSL scripts, `query.lsl` and `touch.lsl`. The first runs queries: you touch to activate it, write the name of an avatar in chat using '/5 firstname lastname', and it replies with the avatar key (after a timeout, the script resets, giving another person a chance to try it out). The second script is to be placed on anything touchable to grab avatar names and keys and send it to your own database. You can, of course, use other methods to grab those; as an exercise, use a Sensor instead, or — even better and consuming much less resources — use a transparent, phantom prim across a place where people have no choice but to go through, and register names & keys as avatars 'bump' into that prim! There are more exotic alternatives, such as registering the name & key when an avatar sits on the prim; or using a llCastRay to figure out if there is anybody nearby. Lots of possibilities! :-) 

## Configuration
You can run the executable either as:

	-port string
        Server port (default "3000")
	-server
        Run as server on port 3000
	-shell
        Run as an interactive shell
     
Basically, if you are running your own server (possibly at home!), you only need to run `gosl-basics -server`. You don't need to set up Apache or nginx or any other third-party software; `gosl-basics` is a fully standalone application and does not depend on anything.

If you're using a shared web server, like the ones provided by [Dreamhost](https://dreamhost.com), then you will very likely want to run `gosl-basics` as a FastCGI application. Why? Well, Dreamhost's Terms of Service explicitly forbid any application to be run all the time (to conserve memory, CPU slices, and, well, open ports). Instead, they offer the ability to run applications as FastCGI applications instead (under their own Apache). This is actually a very cool interface (as opposed to the ancient, non-fast CGI...) allowing parts of the setup of the application to be done when it is called the first time, and then launch requests on demand. _If_ there is a _lot_ of traffic, the application will actually remain active in memory/CPU for a long time! If it only gets sporadic calls once in a while, well, in that case, the application gets removed from memory until someone calls the URL again. I have not tested exhaustively, and this will certainly depend from provider to provider, but Dreamhost seems to allow the application to remain active in memory and in the process space for 30-60 seconds.

Remember to set your URLs on each of the two LSL scripts!

For the standalone server: this will be something like `http://your.server.name:3000/`.

For FastCGI: If your base URL (i.e. pointing to the directory where you have installed `gosl-basics`) is something like `http://your.hosted-server.name/examples` then your URLs will be something like `http://your.hosted-server.name/examples/gosl-basics`. Some providers might force you to add an extension like `.fcgi`, meaning that you will have to remember to add that explicitly every time you compile the application again (or just use a [Makefile](https://www.devroom.io/2015/10/03/a-makefile-for-golang-cli-tools/) for that!).

## Limitations

I found that somehow the standard FastCGI package in Go seems to be limited to just one handler on the path router, so I took that into account and simply tried to figure out what was requested based on the valid parameters (`name` and `key`) passed via GET or POST. A more complex real-world example might require a much more sophisticated router; possibly the [Gorilla](https://github.com/gorilla/mux) router would handle FastCGI better, I don't know.

Note that the current version can be used as a direct replacement for [W-Hat's name2key](http://w-hat.com/#name2key). There is now a 'compatibility mode' with W-Hat: if on the calling URL the extra parameter `"compat=false"` is passed, then cute messages are sent back; if not, then it just sends back the UUID (or the avatar name). Further compatibility with W-Hat's database is not built-in. Note that you need to download W-Hat's database and install it.

Also note that this will work on OpenSimulator grids as well: UUIDs are supposed to be universal and unique, so each avatar on all OpenSimulator grids out there should have their own UUIDs, different from the ones in Second Life®, and you are able to freely mix them together. A better alternative (as an exercise to the user!) would be to capture the grid name from the headers and store it with the avatar name/key pair; in that scenario, you could change the web service so that it only replies with name/key pairs from the grid it has been called from (but storing them all on the same database, with exactly the same interface). This is certainly possible!