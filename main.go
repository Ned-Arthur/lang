package main

import (
	"fmt"
	"log"
	"os"		// File operations
	"strings"
	"slices"
	"strconv"
	//"regexp"
)

/* TODO

Parser
	Deal with spaces within strings

*/


/*** Global State ***/
type progState struct {
	vars map[string]int
	pc int					// Program Counter
	lines []string			// The program stored as an array of lines
}

var state = progState{
	make(map[string]int),
	0,
	make([]string, 0) }

// Regex
//var varName, _ = regexp.Compile("_?[a-zA-Z_]")

// A helper for converting bool comparisons to ints
func Btoi(in bool) int {
	ret := 0
	if in { ret = 1 }
	return ret
}

/*** Helper functions ***/

// Crash in a normal way when we get an error
func die(message string) {
	log.Fatal(message)
}

// Pop the last element from our reverse Polish stack
func pop(stack *[]int) int {
	val := (*stack)[len(*stack) - 1]
	*stack = (*stack)[:len(*stack) - 1]
	return val
}

// Get the last character in a string
func last(str string) byte {
	return str[len(str) - 1]
}

// Search the program for the corresponding 'end' statement
// and skip to the line after it
func traverseTo(token string) {
	for parseLine(state.lines[state.pc])[0] != token { state.pc++ }
	//state.pc++
}

// Search backwards for the start of our block
func traverseBackTo(token string) {
	for parseLine(state.lines[state.pc])[0] != token { state.pc-- }
}

// Convert a raw program line into a list of tokens
func parseLine(line string) []string {
	// Remove all tabs
	line = strings.ReplaceAll(line, "	", "")

	// Split for tokens and remove whitespace only tokens
	toks := slices.DeleteFunc(strings.Split(line, " "),
		func(e string) bool { return strings.TrimSpace(e) == "" })

	return toks
}

func eval(program string) {
	// Split on newlines and filter out empty lines
	state.lines = slices.DeleteFunc(strings.Split(program, "\n"),
		func(e string) bool { return strings.TrimSpace(e) == "" })

	// Evaluate each line
	for state.pc < len(state.lines) {
		toks := parseLine(state.lines[state.pc])

		// Figure out what's going on in this line
		switch toks[0] {
		case "while":
			if evalExpr(toks[1:]) != 0 {
				// Condition is true, keep going
			} else {
				// Condition is false, skip past the end statement and continue executing
				traverseTo("endwhile")
			}
		case "endwhile":
			// Finished one loop of a while, go check the condition again
			traverseBackTo("while")
			continue

		case "if":
			if evalExpr(toks[1:]) != 0 {
			} else {
				traverseTo("endif")
			}
		case "endif":
			// Catch endif if we find one, then keep going
			state.pc++
			continue

		case "print":
			if toks[1][0:1] == "\"" {
				// Found a string
				if string(last(toks[1])) != "\"" {
					die("Unclosed string. You need to close the quotes or remove the spaces")
				}
				fmt.Println(toks[1][1:len(toks[1]) - 1])
			} else {
				// Found an expression to parse
				fmt.Println(evalExpr(toks[1:]))
			}

		// Assume if it isn't a reserved word it's a variable
		default:
			if toks[1] == "=" {
				// In the variable map assign this var to the expression to the right of '='
				state.vars[toks[0]] = evalExpr(toks[2:])
			}
		}

		state.pc++
	}
}

func evalExpr(toks []string) int {
	stack := make([]int, 0)

	// Examine each token and deal with it
	for _, tok := range toks {
		// Handle integers
		if val, err := strconv.Atoi(tok); err == nil {
			stack = append(stack, val)
			continue

		// Handle variables
		} else if val, exists := state.vars[tok]; exists {
			stack = append(stack, val)

		// Handle binary operators
		} else {
			rhs := pop(&stack)
			lhs := pop(&stack)

			// Do action based on operator match
			switch tok {
			// Arithmetic
			case "+":
				stack = append(stack, lhs + rhs)
			case "-":
				stack = append(stack, lhs - rhs)
			case "*":
				stack = append(stack, lhs * rhs)
			case "/":
				stack = append(stack, lhs / rhs)

			// Boolean comparisons
			case ">":
				stack = append(stack, Btoi(lhs > rhs))
			case "<":
				stack = append(stack, Btoi(lhs < rhs))
			case ">=":
				stack = append(stack, Btoi(lhs >= rhs))
			case "<=":
				stack = append(stack, Btoi(lhs <= rhs))
			case "==":
				stack = append(stack, Btoi(lhs == rhs))
			case "!=":
				stack = append(stack, Btoi(lhs != rhs))
			}
		}
	}

	return pop(&stack)
}

func tokenizeProgram(program string) [][]string {
	lines := slices.DeleteFunc(strings.Split(program, "\n"),
		func(e string) bool { return strings.TrimSpace(e) == "" })

	toks := make([][]string, 0)

	for _, line := range lines {
		// Remove all tabs
		line = strings.ReplaceAll(line, "	", "")

		// Split for tokens and remove whitespace only tokens
		lineToks := slices.DeleteFunc(strings.Split(line, " "),
			func(e string) bool { return strings.TrimSpace(e) == "" })

		toks = append(toks, lineToks)
	}

	return toks
}


func main() {
	args := os.Args[1:]
	fPath := args[0]

	dat, err := os.ReadFile(fPath)
	if err != nil {
		panic(err)
	}
	program := string(dat)

	toks := tokenizeProgram(program)

	//eval(program)

	// Dump the program state when we're done
	//fmt.Println(state.vars)
}

