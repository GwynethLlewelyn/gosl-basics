# gosl-basics

A basic example of how to develop external web services for Second Life/OpenSimulator using the Go programming language, using the ever-so-popular `key2name`/`name2key` database as a REST service.

## Why use Go?

This is a basic example (although from the code it might seem complex!) on how to interface an in-world object in Second Life/OpenSimulator with a web application compiled with Go. A huge advantage of Go is in its modular approach, including in the way it resolves dependencies; secondly, it's the easiest cross-compiling language I ever worked with; and, thirdly, it's tremendously efficient.

What this means is that you just need to compile the application and run it — no configuration necessary; no dependencies; no worrying if you have the 'right' version of the language, or the 'right' versions of the libraries and DLLs needed to run it... there is nothing of the sort! You just run the application written in Go and that's it.

Go is a _compiled language designed to be used for system administrators_ (and not necessarily a 'cutey' language for Web designers, no offence meant!). There are no 'virtual machines', no 'interpreters', no 'just-in-time compilers', no 'opcaches', or whatever is currently fashionable to offer in a modern programming language. Go compiles to efficient machine code, end of story. Well, it's not so fast as C (nothing beats C), mostly because it includes a garbage collector. But in a sense it's even stronger-typed than C++ — which aids the compiler to optimize the code — while at the core it's actually a pretty basic programming language. Some even claim that it not even supports "true" object-oriented programming, relying on a different (much simpler and way more efficient) concept instead. The result is that it's easy to learn (perhaps at the same level as C, minus pointers), with little syntactic sugar (no ternary operators, no operator overloading... and even generics were only implemented in recent versions, much against the core developers' original principles) — while insanely powerful to use (at the level of C++ and beyond).

While Go is super-strongly-typed, it actually infers most types automatically. In most cases, you just assign a variable to something; no need to "declare" it first, nor even to declare its type. Go's compiler will figure it out automagically. Thus, you can easily write code that resembles JavaScript — and not TypeScript! — while at the same time benefit from the strict checking done by the compiler.

Here is a basic example:

```go
i := 5
```

This declares a variable, `i`. The `:=` operator is the "assign for the first time" operator. Variables must  only use it once. The compiler automagically figures out that this is a variable assignment, if it's the first time it saw `i`. And since it gets assigned the value `5`, it _must_ be an integer. It cannot be any thing else, and cannot have any other value but an integer.

You cannot redeclare it:

```go
i := 8			// throws error since the variable was already defined.
i := "a string"	// same error; the compiler won't allow you to have any other value type for `i`.
```
Once _defined_, the variable will _always_ have a value. Even if you don't assign anything to it, it'll get a default value (all integers start by being 0, all strings are the empty string ""). There is no ambiguity. You don't need ever to worry if your variable is "unknown", or "unassigned", or pointing to null. The compiler either sees a variable being _correctly_ defined (there are many ways to do so, but only a _limited_ number of ways, all very-well known) or throws an error otherwise. From then onwards, you cannot ever change its type. It's written in stone and you can be sure that it won't change under the hood.

You also can't assign things to it that are not of its type:
```go
i = 2.3		// wrong, 2.3 is a float, not an integer.
i = "what?"	// wrong, `i` is an integer, you cannot assign strings to it!
```
That should be obvious for anyone used to strongly-typed languages. But because Go code so often looks as if the programmer has forgotten to declare variables, it _almost_ looks like you can have the sort of volatile variables so popular under PHP or JavaScript or even Lua (and many other interpreted language).

This strictness _also_ applies to pointers. You _can_ have pointers — since everything is passed by value, it's useful to have a way to pass pointers to a value instead, mostly to save memory. But you cannot wreak havoc with pointers — they are as strongly typed as any other type! A pointer to an integer is _not_ a pointer to a float — even if, when looking inside the actual pointer representation, they seem to be the same thing (basically a 64-bit integer). But they aren't, and the compiler _knows_ they're different, so it does _not_ allow you to mix and match them. A "pointer to a type" is _also_ a type, and as strongly enforced as any other.

Well, if you hate too-strongly-typed languages (think C#!) because they force you to spend most of your time converting between almost-the-same-but-not-quite types, endlessly adding new classes to deal with the mess of having similar but not equal types... good news: Go is totally different in concept. In practice, the _need_ to explicitly typecast variables all the time is severely restricted to a few exceptional cases, and that's because, _most_ of the time, you will _not_ need to worry about any of that. In Go, the most interesting underlying principle is "if it walks like a duck, and quacks like a duck, then I'll be happy to consider it a duck". That's actually how Go implements its own flavour of class inheritance: so long as your function "walks and quacks like a duck" (i.e., it has the same _signature_ as a duck), you can use it with the "duck class" — you don't need to create a new class, inherit a new class, write conversion functions, have special getters/setters or similar syntactic sugar to make such conversions easier, etc. You just rely on the principle that ducks walk in a certain way and quack. Simple. The compiler _knows_ that if something walks like a duck but does *not* quack, but clucks... well, then, it's _not_ a duck, and it throws a compilation error. The compiler _knows_ that a duck is not a chicken (and vice-versa) and keeps them apart.

Go has also a weird look to its code density: it's simultaneously a very compact language (you write little to achieve a lot!), but at the same time, it _seems_ to be as verbose as, well, BASIC. There are almost none of those fancy things that are so popular on contemporary modern languages. I've already mentioned the lack of a ternary operator, but there are many more cases of (apparently) missing functionality.

The first that stands out is the complete absence of a `try...catch` construction. Error handling _seems_ to be insanely primitive without those fancy constructs (which vary across languages, of course — some are more sophisticated in the way errors are handled, with multiple possible outcomes for an assertion test), and, in fact, Go code is full of entries that look much more like BASIC than a "true" descendant of C/C++/C#/Java/JavaScript/PHP family.

However, don't get deluded by its _apparent_ simplicity. In fact, idiomatic Go is based on the assumption that most functions will throw an error (which, BTW, is just another type, nothing special about it; and a pretty primitive one at that!). You essentially check if the error is nil or not. If it is, that means "no error" and you can go ahead. If it's anything else, then it contains some sort of error. Good, but, in Go, you don't really need to immediately jump on it. You can just pass it along to whoever called _your_ code. This is actually baffling for new users (including myself). There is not really a `throw` as you do on other languages, but rather simply saying, "oh goodie, I got an error, let's return it and forget about the rest of our code". And that's it.

Well, almost. Go has two tricks up its sleeve. The first is a Lua-like way of returning _more than one value_. While strange at the beginning, once you get used to it, you don't want to go back to any other single-returned-value language any longer. The idea is that functions don't simply return a null (or nil in Go-parlance) and expect programmers to `catch` whatever error was produced; instead, Go functions tend to return _both_ the result _and_ the error (or nil for no error) — as two _separate_ variables, which you can then proceed to independently check:

```go
if result, err := myFunction("some string parameter", 1337); err != nil {
	fmt.Println("Got error:", err)
}

Note that `if` can also take _several_ statements, not just one; the above could be written as

```go
if result, err := myFunction("some string parameter", 1337)
if err != nil {
	fmt.Println("Got error:", err)
}
```
But that level of "verbosity" can be condensed as shown above, which has an added advantage: `err` is just declared on a _local_ scope, and will be unavailable outside it; by contrast, in the block below, a new `err` variable will be defined and will remain in existence after the check.

An interesting feature of Go is that you can also "catch" errors from functions called — several levels deep, in some cases, if all levels implement things correctly. For instance, consider that you have a function that reads some configuration parameters from a file. But if that file doesn't exist, or is not readable (permission error!), what can you do about it? Well, idiomatic Go simple calls a `panic` — and passes the error along to whoever called it. The caller can then make a choice: will it ignore the error — or attempt to provide a fix? In the given example, you might have a "fall-back" file to read from. Or forget about all the missing/broken/inacessible flle and plod along the rest of the code. All these approaches are perfectly acceptable: you can decide to _consume_ the panic error, while not really "crashing" to disk.

\[To be continued...\]

Finally, because Go effectively comes with its own webserver... the code might be a little more lengthy than what you'd expect for a simple thing like this. But I've been mean and added a lot of bells and whistles, so that you don't get merely a 'skeleton' project with no clue how to extend it. `gosl-basics` may be basic, but it's a fully-fledged web server, able to run as either a stand-alone server or as a FastCGI application, including an interactive shell (or command line) to do some quick testing, it parses command-line arguments, and has a fairly sophisticated logging system, which will do automatic rotation and so forth... all the tools that you need to build a 'serious' application, in fact. In case you are wondering: no, I didn't develop any of those features, they are all shared by Go programmers via open-source repositories, which extend the 'core' functionality of Go (meaning the libraries provided by the original developers and maintainers) in a very easy way.

## Yes, fine, but what does this application actually _do_?

I could have provided a very basic 'echo' service, which is what 99.9% of people do when demonstrating new features in a web service; but I found out from experience that such an 'echo' service almost always uses nasty tricks, because it's so simple, to make the code minimalistic. And then it's actually very hard to figure out how to use it in a 'real world' situation!

Instead, I've made another version of the (in)famous **name2key** database. If you're a LSL scripter, you're aware that Linden Lab provides a way to get the name of an avatar (or any object, really) by supplying its UUID (or key), but the reverse is not true. Why? Because it could be abused — if you have the avatar keys for _all_ the avatars, then you could theoretically spam them all with instant messages, and there would be no way for them to 'unlist' themselves from such a spam list. So Linden Lab deliberately never provided that functionality. The only alternative is to patiently grab names & keys from avatars who pass by a sensor, touch an object, sit on an object, and so forth, and accumulate those names and keys on an external database (which will have literally millions of entries). Several people have done such databases in the past, and I still have one (running under PHP) which also served as a purpose to illustrate the ability that Second Life has to store persistent information in external servers. That was more than a decade ago!

The most famous — and most complete! — name2key database had been compiled by the (in)famous [W-Hat group](http://w-hat.com/). I'll leave as an exercise for you to look them up and their history. They have kindly left their service behind, and even given away their source code, placing it in the public domain. There is a catch, though: they fancy an obscure programming language called 'Clojure', which is basically LISP compiled to a Java Virtual Machine. LISP is the kind of language that only ubergeeks know; developed in 1958 (yes, seriously!), it was the first programming language for artificial intelligence. It still has lots of uses, even outside the field of AI (the famous computer-aided design platform AutoCAD used a dialect of LISP for ages to extend its functionality), but it's something that most people will never encounter in their lives :-) Now I have nothing against LISP (it used to be one of my favourite languages), but I hate the idea that I need to install a whole Java environment, plus configuring Clojure properly, just to be able to use their (simple) code.

Instead, I rather prefer to show how it's done in Go.

## Installation overview

Requirements:

-   You need to have [Go](https://golang.org) installed and configured properly
-   Your computer/server needs to have a publicly accessible IP (and set it up with a **dynamic DNS provider** such as [no-ip.com](https://www.noip.com/remote-access)); you _can_ run it from home if you wish

Finally, `go get git.gwynethllewelyn.net/GwynethLlewelyn/gosl-basics.git`, and you _ought_ to have a binary executable file in `~/go/bin` called `gosl-basics.git`. Just run it!

Then grab the two LSL scripts, `query.lsl` and `touch.lsl`, which ought to be in `~go/src/git.gwynethllewelyn.net/GwynethLlewelyn/gosl-basics`. The first runs queries: you touch to activate it, write the name of an avatar in chat using '/5 firstname lastname', and it replies with the avatar key (after a timeout, the script resets, giving another person a chance to try it out). The second script is to be placed on anything touchable to grab avatar names and keys and send it to your own database. You can, of course, use other methods to grab those; as an exercise, use a Sensor instead, or — even better and consuming much less resources — use a transparent, phantom prim across a place where people have no choice but to go through, and register names & keys as avatars 'bump' into that prim! There are more exotic alternatives, such as registering the name & key when an avatar sits on the prim; or using a llCastRay to figure out if there is anybody nearby. Lots of possibilities! :-)

If something went wrong, you might need to download all external packages manually. Theoretically, the `go get` command is supposed to be clever enough to figure out everything it needs and download _everything_ automatically, but we all know how complex systems manage to fail, don't we? So, if needed, these are all the external packages (i.e. not part of the standard library) to be downloaded:

```sh
go get github.com/dgraph-io/badger/v4
go get github.com/h2non/filetype
go get github.com/h2non/filetype/matchers
go get github.com/op/go-logging
go get github.com/spf13/pflag
go get github.com/spf13/viper
go get github.com/syndtr/goleveldb/leveldb
go get github.com/syndtr/goleveldb/leveldb/util
go get github.com/tidwall/buntdb
go get gopkg.in/natefinch/lumberjack.v2
```

These are for 3 different key/value databases; handling command-line flags and a configuration file; a powerful logging system; and a way to rotate logs (a lumberjack... chops down logs... get it? :) Aye, Gophers have a sense of humour).

**Note:** If modules are fully functional on your system, you ought to type this instead:

```sh
go mod tidy
go get -u
```

## Configuration

You can run the executable with (at least) the following parameters:

  -b, --batchblock int    How many entries to write to the database as a block; the bigger, the faster, but the more memory it consumes. (default 100000)
	  --config string     Configuration filename [extension defines type, INI by default] (default "config.ini")
	  --database string   Database type [badger | buntdb | leveldb] (default "badger")
  -d, --debug string      Logging level, e.g. one of [DEBUG | ERROR | NOTICE | INFO] (default "ERROR")
	  --dir string        Directory where database files are stored (default "slkvdb")
  -i, --import string     Import database from W-Hat (use the csv.bz2 versions)
  -l, --loopbatch int     How many entries to skip when emitting debug messages in a tight loop. Only useful when importing huge databases with high logging levels. Set to 1 if you wish to see logs for all entries. (default 1000)
	  --nomemory          Attempt to use only disk to save memory on Badger (important for shared webservers) (default true)
  -p, --port string       Server port (default "3000")
	  --server            Run as server on port 3000
	  --shell             Run as an interactive shell

Basically, if you are running your own server (possibly at home!), you only need to run `gosl-basics --server`. You don't need to set up Apache or nginx or any other third-party software; `gosl-basics` is a fully standalone application and does not depend on anything.

If you're using a shared web server, like the ones provided by [Dreamhost](https://dreamhost.com) or [Bluehost](https://bluehost.com), then you will very likely want to run `gosl-basics` as a FastCGI application. Why? Well — to take an example — Dreamhost's Terms of Service explicitly forbid any application to be run all the time (to conserve memory, CPU slices, and, well, open ports). Instead, they offer the ability to run applications as FastCGI applications instead (under their own Apache). This is actually a very cool interface (as opposed to the ancient, non-fast CGI...) allowing parts of the setup of the application to be done when it is called the first time, and then launch requests on demand. _If_ there is a _lot_ of traffic, the application will actually remain active in memory/CPU for a long time! If it only gets sporadic calls once in a while, well, in that case, the application gets removed from memory until someone calls the URL again. I have not tested exhaustively, and this will certainly depend from provider to provider, but Dreamhost seems to allow the application to remain active in memory and in the process space for 30-60 seconds.

Remember to set your URLs on each of the two LSL scripts!

For the standalone server: this will be something like `http://your.server.name:3000/`.

For FastCGI: If your base URL (i.e. pointing to the directory where you have installed `gosl-basics`) is something like `http://your.hosted-server.name/examples` then your URLs will be something like `http://your.hosted-server.name/examples/gosl-basics`. Some providers might force you to add an extension like `.fcgi`, meaning that you will have to remember to add that explicitly every time you compile the application again (or just use a [Makefile](https://www.devroom.io/2015/10/03/a-makefile-for-golang-cli-tools/) for that!). Also, if you wish to run it automatically when the server starts, you can take a look at some [quick & dirty instructions](startup-scripts/README-systemd.md) for doing so.

Note that the first time ever the application runs, it will check if the database directory exists, and if not, it will attempt to create it (and panic if it cannot create it, due to permissions — basically, if it can't create a directory, it won't be able to create the database files either). You can define a different location for the database; this might be important when using FastCGI on a shared server, because you might wish to use a private area of your web server, so that it cannot be directly accessed.

The `--shell` switch is mostly meant for debugging, namely, to figure out if the database was loaded correctly (see below) and that you can query for avatar names and/or UUIDs to see if they're in the database. Remember to run from the same place where the database resides (or pass the appropriate `--dir` command). Also, you will get extra debugging messages.

The `--nomemory` switch may seem weird, but in some scenarios, like shared servers with FastCGI and using the Badger database, the actual memory consumption may be limited, so this attempts to reduce the amount of necessary memory (things will run much slower, though; the good news is that there is _some_ caching).

See below for instructions for importing CSV bzip2'ed databases using `--import`. The CSV file format is one pair **UUID,Avatar Name** per line, and all of that bzip2'ed.

## Limitations

I found that somehow the standard FastCGI package in Go seems to be limited to just one handler on the path router, so I took that into account and simply tried to figure out what was requested based on the valid parameters (`name` and `key`) passed via GET or POST. A more complex real-world example might require a much more sophisticated router; possibly the [Gorilla](https://github.com/gorilla/mux) router would handle FastCGI better, I don't know.

Note that the current version can be used as a direct replacement for [W-Hat's name2key](http://w-hat.com/#name2key). There is now a 'compatibility mode' with W-Hat: if on the calling URL the extra parameter `"compat=false"` is passed, then cute messages are sent back; if not, then it just sends back the UUID (or the avatar name). Further compatibility with W-Hat's database is not built-in.

To actually _use_ the W-Hat database, you need to download it first and import it. This means using the `--import` command (use the `name2key.csv.bz2` version). W-Hat still updates that database daily, so, with some clever `cron` magic, you might be able to get a fresh copy every day to import. Note that the database is supposed to be unique by name (and the UUIDs are not supposed to change): that means that you can import the 'new' version over an 'old' version, and only the relevant entries will be changed. Also, if you happen to have captured new entries (not yet existing on W-Hat's database) then these will _not_ be overwritten (or deleted) with a new import. To delete an old database, just delete the directory it is in.

Importing the whole W-Hat database, which has a bit over 9 million entries, took on my Mac 1 minute and 38 seconds. Aye, that's quite a long time. On a shared server, it can be even longer. The code has been substantially changed to use `BatchSet` which is allegedly the recommended way of importing large databases, but even in the scenario to consume as little memory as possible, it will break most shared servers, simply because Go's garbage collector will not be fast enough to clean up after each batch is sent — I may have to take a look at how to do this better, perhaps with less concurrency.

This also works for OpenSimulator grids and you can use the same scripts and database if you wish. Currently, the database stores an extra field with UUID, which is usually set to the name of the grid. Linden Lab sets these as 'Production' and 'Testing' respectively; other grid operators may use other names. There is no guarantee that every grid operator has configured their database with an unique name. Also, because of the way the key/value database works, it _assumes_ that all avatar names are unique across _all_ grids, which may not be true: only UUIDs are guaranteed to be unique. The reason why this was implemented this way was that I wanted _very fast_ name2key searches, while I'm not worried about very slow key2name searches, since those are implemented in LSL anyway. To make sure that you can have many avatars with the same name but different UUIDs, well, it would require a different kind of database. Also, what should a function return for an avatar with three different UUIDs? You can see the problem there. Therefore I kept it simple, but be mindful of this limitation.

Oh, and with ~10 million entries, this is slow.

## Which database is best?

This is an ongoing debate, as each K/V database developer publishes their own benchmarks to 'prove' their solution is 'best' (or faster). In my personal scenario, I was looking for a solution that worked best on a shared server a with small memory footprint and limited CPU resources, such as provided by Dreamhost. In fact, Dreamhost's watchdog had to forcefully kill so many instances of this application while testing it with different databases that I got a notification from their automated system saying that I was 'consuming too many resources'!

BoltDB is probably the best-known fastest K/V database for Go (and I'm pretty sure it was named after Usain Bolt...). It assumes that the database is stored in SSD disks (not spinning disks) and/or memory, and that's where its speed comes from. It is based on the LMDB K/V database, but focused on presenting an easy-to-use interface instead of raw performance. It is a fully ACID-compliant database with transactions.

Badger has been developed by Google and is currently undergoing a major rewriting of its API so that it ultimately becomes a drop-in replacement for BoltDB; this has meant a considerable amount of changes in the API as the developers revamp the code and do lots of API calls entirely from scratch. In fact, the major reason for `gosl-basics` not to compile will very likely be due to yet another change of Badger's API. Badger, like BoltDB, has also been fine-tuned to give the best possible performance using SSD disks, and it goes even further than BoltDB in terms of conceptual architecture so that it achieves better performance. Like BoltDB, it also uses transactions (the API for those has changed considerably over time!) and allegedly it's also fully ACID-compliant. The current version of Badger has a lot of parameters that can be tweaked with in order to achieve better performance according to the kind of data that is stored on the K/V database.

Finally, LevelDB is a 'baseline' K/V database, in the sense that it was originally developed by Google and written in C, but, over the years, it has been ported to other languages from scratch, including Go. It has a very simple and understandable API (way easier to figure out for an amateur programmer such as myself!), it was developed with spinning disks in mind, and, unlike the other two, transactions are optional (thus, you can use LevelDB in a non-ACID compliant way), and they have also additionally implemented something they call 'batches', which are not really 'transactions', just a way to execute a series of commands in a more efficient way. It relies a bit more on disk than on memory (in contrast to the other two databases) and allegedly is supposed to be the 'worst' of the three, in the sense that it provides a 'baseline' performance, against which all other K/V database engines can be compared with.

All three are embedded, which has its set of advantages and disadvantages, but in the current scenario (getting UUID keys for avatars) it made more sense to pack everything into a single binary. My previous solutions used a PHP frontend to a MySQL database backend, where the actual searching was performed.

Now, which one is 'best'? Honestly, I have only tried them in two setups: on my MacBook Pro from mid-2014, which already features a SSD disk; and on two remote Linux server with spinning disks, one of which (the one hosted at Dreamhost) is a shared server with memory and CPU limitations (namely, their watchdog will kill any process consuming much more than 280 MBytes of RAM); the other is my own 'bare metal' server which has no such limitations. I did not run any real, statistically significant benchmarks, but, if you run this application, you will see from the logs that the main operations (loading the W-Hat database, searching for avatar given an UUID, and searching an UUID giving an avatar name) are being measured by the code. And what was interesting to notice was that LevelDB performed _far better_ than the other two, by at least two orders of magnitude, _even on a MacBook with a SSD disk and unlimited memory for the process to run_ (well, the limit being the available RAM + swap disk). This completely baffled me and made me scratch my head and ask the Badger developers for advice in fine-tuning. Still, even though I have provided two different memory footprint configurations — one thought to be better for environments with limited resources — the truth is that neither Badger nor BoltDB come even close to LevelDB's performance. In particular, it's impossible to load the full W-Hat database with its 9 million records at Dreamhost; and even if I do the simple trick to load it on the Mac (or on one of my servers) and copy it over to Dreamhost, the FastCGI application will fail simply with initialising the database to deal with the 9 million records _even if they are all stored on disk_.

Their behaviour is also erratic. There will be a more thorough discussion on a blog post I'm writing, but, in spite of forcefully running Go's garbage collector when importing the file (after each batch or set is loaded, memory is released), both BoltDB and Badger will grow the allocated memory to increasingly higher values, until eventually they consume pretty much all available memory. I speak of an 'erratic' behaviour because it's not easy to understand what is going on: sometimes the memory growth is linear, it reaches a plateaux for a while, and then grows again; sometimes, a little bit of memory is released to the system, just to be swallowed up in huge chunks after a few dozens of thousands of records are imported. At the end of the day, both will ultimately break the 280 MByte RAM limit after importing perhaps a million records (sometimes less); under the Mac, they will eagerly consume a couple of GBytes to complete the import.

Now, one might argue that all this is due to Go's notoriously bad garbage colector (aye, with each release, it becomes better, but it's still a pain to deal with). But that argument cannot be universally valid. The Go compiler itself runs quite well even under Dreamhost's constricted environment; so it is certainly releasing enough memory to keep it well under Dreamhost's limit. And this is where LevelDB shines: you can see it allocating a linear amount of memory, enough to deal with one batch; and then everything is properly released; and when the next batch is loaded, it has allocated _exactly_ the same amount as before: thus, it's absolutely predictable how much memory it will allocate each time, and it's a linear function correlating with the size of the data actually being loaded on the K/V database (in my scenario, the Avatar Name, the UUID, the grid it comes from — perhaps just a bit under 200 bytes per entry!).

So what is going on with Badger and BoltDB? Well, both have a 'compacting' algorithm — a separate thread which runs through the whole structure and tries to 'compact' the data as it is being written and/or deleted. My assumption (not being familiar with the code, even though it's open source and some of it relatively easy to understand) is that the compaction thread is playing havoc with Go's garbage collector — i.e. there is a race condition between both, the GC knowing that I've flagged some memory to be released, but the compacting thread 'stealing' some of that memory to look at freshly-written data to see if it can be further compacted. There might be a way to disable the compacting threads — at least under Badger this seems to be the case, but when I tried to do that, the application panicked instantly... — or fine-tune them in some way (trying to avoid allocating so much memory, for instance). But clearly neither solution has been designed for a very memory/CPU-constrained scenario; they achieve performance by assuming that almost everything is always loaded in RAM, and the rest comes from a SSD drive which can be instantly accessed.

That explanation might make sense for _importing_ data (because that means that the K/V database is being written all the time), but it fails to explain why a simple _search_ requires that much memory (since nothing is being written, the compacting algorithm should not be running, etc.). Well, I can understand that some elements of the K/V database — all of which have a tree of some sort, usually a B+ tree or a LSM tree, or a variant — are being loaded on RAM first, and that means allocating enough RAM for 9 million records. Both Badger and BoltDB also have 'caches' for repeated queries, but in my experience, the time gained by reading from the cache is negligible compared to a 'fresh' query. And while all three feature 'iterators' — a way to go through the whole tree until a match is found, which is important for retrieving avatar names based on UUID — BoltDB and Badger are simply terrible at doing that: even on the Mac, with unlimited RAM and a SSD disk, they take almost a minute (half a minute if cached...) to go through all 9 million entries! That's certainly unacceptable, even though I'm perfectly aware that the correct solution would be to have _two_ databases, one for avatar name -> UUID, and the other from UUID -> avatar name; but for the purpose of this application I wanted to show how to iterate through a K/V database, as opposed to illustrate the _optimal_ method of using it.

Last but not least, I didn't bother to use transactions with LevelDB; this _might_ make a huge difference. Or perhaps not. The first version of Badger I've used (as said, Badger's API is in constant mutation) did also make transactions optional, and I didn't see any performance difference between using transactions or not using them (which, in a sense, shows how optimised the code is, i.e. using transactions does not make it visibly slower, and, naturally enough, makes all operations much safer). Badger also seems paranoid in making sure that data is _never_ lost, using an internal versioning system (I'm afraid I didn't delve much into it). All of this _might_ contribute for its slowness, and, since it's actively looking up at BoltDB as its direct 'competitor', it's not hard to see how both are constantly copying ideas from each other.

LevelDB, by contrast, doesn't bother with any of that. It seems to achieve sheer performance by keeping everything simple. I mean, I'm not arguing about a 5% increase in performance (which would be more than enough to become a subject of an academic paper in fine-tuning K/V databases); nor even about the way LevelDB manages to keep a small memory footprint even when importing 9 million records. No, I'm really talking about _orders of magnitude_. A search under BoltDB or Badger takes something between 50 and 500 ms; a full iteration, as said, can take as long as one full _minute_. Importing 9 million records takes around 3 minutes.

Contrast this with LevelDB: a search takes _micro_seconds, not \_milli_seconds! That's pretty much what I \_expect_ from a K/V database; after all, I can certainly get queries in the low millisecond range using, say, MySQL. A full iteration through all records takes of course much longer: a couple of seconds or so. Well, for the purposes of demonstration, having a round-trip time of 2 seconds is almost tolerable within Second Life. One minute, or even half a minute, is not.

So clearly either my code is seriously wrong, or I have no idea how people benchmark BoltDB and Badger against LevelDB! :-)

## Real-world scenarios

This whole application is supposed to be an _exercise_, _not_ a fully-working, optimal solution for the `name2key` issue (there is already W-Hat's interface for that!). My own PHP + MySQL solution could achieve results pretty close to what Go does in this scenario. I'm even aware that, because W-Hat's database export comes as a _sorted_ file, I might achieve far better results if I merely did some binary searches directly on the disk file! (An exercise left for the user; to make it harder: try to do the same inside the bzip2'ed file — it's not as hard as it seems programatically, but the performance hit might be huge!)

As said at the beginning, my purpose here was to demonstrate a 'real' application written in Go, which interfaces with Second Life, as opposed to a 'cute' example (like an infamous 'echo' application which usually is way too hard to figure out how to use that as a starting point for a 'real' application). For that, I really went overboard with unnecessary bells & whistles such as reading command-line parameters and/or configuration files (which can even be changed in real-time and will immediately be read by the application). Because I was told to use the Badger K/V database for this kind of solution, I thought that I ought to try it out, as opposed to simply loading everything in memory, and/or doing direct binary searches on the W-Hat file; because the results I got with Badger were frankly disappointing, I explored two other alternatives, and, at least in my own scenario, LevelDB works best and fastest, but your own mileage may vary.

Also note that there are several LevelDB implementations in Go; it's a very popular K/V database in other programming languages, and it's well-documented, so naturally different people tried to attempt to port it to Go. My choice was mostly influenced by how often this particular package had been updated and maintained recently. And there are plenty of other K/V databases available for Go, as well as some NoSQL databases, and so much more. I might do a version in the future using an external database such as MySQL (or SQLite, although I had some issues with concurrency when using the most favoured SQLite package for Go), but doing versions for each and every kind of database package out there requires way more time than I currently have.

Still, I hope I gave you at least a hint of how web services are provided using Go, and how such web services can be used for Second Life. Needless to say, I've drawn a lot of inspiration for this application from my own code currently being used in my PhD dissertation; but most of it is actually written from scratch (mostly because of copyright issues!) and bears only a marginal resemblance to that code (in the sense that the same person, writing the same code twice, will naturally write it in a similar style!). The reverse was also true, while exploring some aspects of the Go packages used in this application, I also re-used it for other projects. That's the whole point of making this open source and available under a very permissible MIT licence!
