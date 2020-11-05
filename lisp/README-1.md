# Writing a Clojure Interpreter in Python & Go Part I

### Introduction

There's an old joke linguistics professors tell (just setting your expectations here):

> During the cold war, the US developed a system to translate from Russian to English and back.
> When they finished writing the system, they decided to test it by giving it a sentence in English, translating it to Russian and back.
> They gave it the sentence "The spirit is willing but the flesh is weak." And got back "The vodka is good but the meat is rotten."

The point of this joke is to show that different languages have a different way of saying things. This is true for programming languages as well.

Working with Python for 25 years and with Go for the past 10, I consider myself fluent in both languages. In Python we say you write "pythonic" code when you grok the language and in Go we say you write "idiomatic Go".

I'd like to explore and compare both languages. I believe that programming languages are tools and that "If all you have is a hammer, every program looks like a nail."

_Note: Except for C++ where every problem looks like your thumb :)_

We're going to compare both languages by ... implementing an interpreter to another language. During the development of the interpreter we'll discuss syntax, types, scope and many other aspects that related to programming language design and implementation.

### The Target Language

I'd like to focus the discussion on language design and semantics, so I've picked a language that's easy to parse: Clojure. Clojure is a dialect of lisp, it has some interesting ideas on concurrency and is a fun language to code in.

Note: Clojure runs on the JVM, but there’s an implementation of it in Go called [joker](https://joker-lang.org/).

Here's an example:

**Listing 1: Example Clojure Program - collatz.clj**

```
01 (def collatz-step
02   (fn [n]
03     (if (= (mod n 2) 0)
04 	(/ n 2)
05 	(+ (* n 3) 1))))
06 
07 (println (collatz-step 7)) ; 22
```

Listing one shows the Clojure code to calculate a step in the [Collatz conjucture](https://en.wikipedia.org/wiki/Collatz_conjecture). On line 1 we use the `def` keyword to assign a value to the name `collatz-step`. On line 2 we define the value which is a `fn` expression - an anonymous function. On line 3 we have an `if` statement and on line 4 we have the true branch of the `if` and on line 5 we have the false branch of the `if` statement. On line 08 we use the `println` function to print the result of calling `collatz-step` on the number 7 - which outputs `22'.

Clojure programs are built like an expression tree and use prefix operators, meaning we write `(+ 1 2)` vs `1 + 2`. There's also no `return`, the return value is the last expression in the function.

![](https://imgs.xkcd.com/comics/lisp_cycles.png)


### Python & Go Implementation

**Listing 2: Collatz step in Go - collatz.go**
```
01 package main
02 
03 import (
04 	"fmt"
05 )
06 
07 func collatzStep(n int) int {
08 	if n%2 == 0 {
09 		return n / 2
10 	}
11 	return n*3 + 1
12 }
13 
14 func main() {
15 	fmt.Println(collatzStep(7)) // 22
16 }
```

Listing 2 shows the implementation of `collatz` in Go.

**Listing 3: Colltaz step in Python - collatz.py**

```
01 def collatz_step(n):
02     if n % 2 == 0:
03         return n // 2
04     return n * 3 + 1
05 
06 
07 print(collatz(7))  # 22
```

Listing 3 shows the implementation of `collatz` in Python.

### Comparison

Let's go over some similarities and differences between the languages.

#### Source Code

All three languages use files to store source code. This is an approach that most programming languages take, but isn't the only one. Who says that code structure should be tied to the file system? There are visual languages like [MIT Scratch](https://scratch.mit.edu/) and there are languages like [Smalltalk](https://en.wikipedia.org/wiki/Smalltalk) where the code is stored ... somewhere in the image.

Another design decision is what can be an identifier (name) in the language. Go & Python both take a similar approach and Clojure takes a wider stance. `collatz-step` is not a valid identifier in Python or Go. Both Python & Go allow Unicode identifiers (e.g. `π = 3.14`), our Clojure implementation will take the same approach. 

#### Syntax

Go takes after C, with it's infix notation (`1 + 2`) and using curly braces for scope. Python also uses infix notation but uses indendentation and `:` for scope. Clojure, uses prefix notation `(+ 1 2)` and uses braces for lists.

#### Execution

Python & Clojure are interpreted, the interpreter will build an abstract syntax tree (AST) and then evaluate it. Python compiles the AST to byte code and runs this byte code in a virtual machine. Go compiles it's programs to AST and then to machine code - there's no runtime involved.

_Note: Go does compile parts of its runtime, like the garbage collector, into the executable. But it doesn't need a VM to interpret instructions to execute._

### Conclusion

By looking at the differences between programming languages, you can understand each language better and clearly see the design trade offs every language makes.

Implementing your own language is a right of passage for developers, and also as Steve Yegge [said](http://steve-yegge.blogspot.com/2007/06/rich-programmer-food.html)

> If you don't know how compilers work, then you don't know how computers work.

This series of posts is inspired by Peter Norvig's [(How to Write a (Lisp) Interpreter (in Python))](https://norvig.com/lispy.html). In the next part we'll start implementing our Clojure interpreter both in Go & Python.


