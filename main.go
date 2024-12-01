package main

import (
    "fmt"
    "log"
    "os"
    "strings"
    "slices"
    "strconv"
    "bufio"
)

/* TODO

Parser
    Identify tokens not seperated by a space

*/

/*** Constants and Enums ***/

type Type int

const (
    Int Type = iota
    String
)

/*** Global State ***/
type scope struct {
    ints map[string]int
    strings map[string]string
}

func newScope() scope {
    return scope{ make(map[string]int), make(map[string]string) }
}

// Data for function calls: where to return to and scoped variables
type functionCall struct {
    returnLine int
    funcScope scope
}

type progState struct {
    tokens [][]string                   // The program stored as a 2d token array
    globScope scope                     // Store variables
    funcs map[string]functionSignature  // Store function definition data
    pc int                              // Program Counter
    functionCalls []functionCall        // An array of data for function calls
    executingFunction bool
    parsingFunCallArgs bool             // Flag for when we need to look at the *previous* stack frame
    returnValue int                     // What the last function returned
}

var state = progState{
    make([][]string, 0),
    newScope(),
    make(map[string]functionSignature),
    0,
    make([]functionCall, 0),
    false,
    false,
    0 }

type argument struct {
    argName string
    argType Type
}

type functionSignature struct {
    startline int               // pc value for function body
    args []argument             // Arg name : Arg type eg. "count":Int
}


// Get the current scope: either global or in the current function
func getWorkingScope() scope {
    workingScope := state.globScope
    if state.executingFunction {
        workingScope = last(&state.functionCalls).funcScope
    }
    return workingScope
}


/*** Helper functions ***/


// A helper for converting bool comparisons to ints
func Btoi(in bool) int {
    ret := 0
    if in { ret = 1 }
    return ret
}

// Pop the last element from a stack
func pop[T any](stack *[]T) T {
    val := (*stack)[len(*stack) - 1]
    *stack = (*stack)[:len(*stack) - 1]
    return val
}

func last[T any](stack *[]T) T {
    val := (*stack)[len(*stack) - 1]
    return val
}

// Get the last character in a string
func lastChar(str string) byte {
    return str[len(str) - 1]
}

// Strip the quotes from a string token
func stringContents(str string) string {
    return str[1:len(str) - 1]
}

// Search the program for the corresponding 'end' statement
// and skip to the line after it
func traverseTo(token string) {
    for state.tokens[state.pc][0] != token {
        state.pc++
        if len(state.tokens) == state.pc {
            log.Fatalf("Didn't find corresponding end statement: %s", token)
        }
    }
}

// Search the program for the corresponding end statement, but if we find an
// alternative stop there
func traverseToButCatch(token string, altToken string) {
    for state.tokens[state.pc][0] != token {
        if state.tokens[state.pc][0] == altToken {
            return
        }

        state.pc++
        if len(state.tokens) == state.pc {
            log.Fatalf("Didn't find corresponding end statement: %s", token)
        }
    }
}

// Search backwards for the start of our block
func traverseBackTo(token string) {
    for state.tokens[state.pc][0] != token { state.pc-- }
}

// Turn a raw file string into a 2d token array
func tokenizeProgram(program string) [][]string {
    lines := slices.DeleteFunc(strings.Split(program, "\n"),
        func(e string) bool { return strings.TrimSpace(e) == "" })

    toks := make([][]string, 0)

    for _, line := range lines {
        // Remove all tabs
        line = strings.ReplaceAll(line, "\t", "")

        // Split for tokens and remove whitespace only tokens
        lineToks := slices.DeleteFunc(strings.Split(line, " "),
            func(e string) bool { return strings.TrimSpace(e) == "" })

        if len(lineToks[0]) >= 2 && lineToks[0][0:2] == "//" {
            continue
        }

        processedToks := make([]string, 0)

        i := 0
        for i < len(lineToks) {
            tok := lineToks[i]

            // Handle Strings
            if tok[0:1] == "\"" {
                for lastChar(tok) != "\""[0] {
                    i++
                    tok += " " + lineToks[i]
                }
            }

            processedToks = append(processedToks, tok)
            i++
        }

        toks = append(toks, processedToks)
    }

    return toks
}

// Evaluate the program as a 2d token array
func eval(tokens [][]string) {
    state.tokens = tokens
    // Evaluate each line
    for state.pc < len(tokens) {
        lineToks := tokens[state.pc]
        // Figure out what's going on in this line
        switch lineToks[0] {
        case "while":
            if evalExpr(lineToks[1:]) != 0 {
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
            if evalExpr(lineToks[1:]) != 0 {
            } else {
                traverseToButCatch("else", "endif")
            }
        case "else":
            traverseTo("endif")
        case "endif":
            // Catch endif if we find one, then keep going
            state.pc++
            continue

        // Function defenitions
        case "void", "int":
            name := lineToks[1]

            if len(lineToks) <= 2 || lineToks[2] != "(" || lineToks[len(lineToks) - 1] != ")" {
                log.Fatal("Function definition without arguments")
            }

            // Push to func map
            state.funcs[name] = functionSignature{ state.pc, make([]argument, 0) }
            
            for i := 3; i < len(lineToks) - 1; i += 3 {
                if lineToks[i + 2] != "," && lineToks[i + 2] != ")" {
                    log.Fatal("Missing comma in argument list @i=", i)
                }
                argName := lineToks[i + 1]
                argType := Int
                if lineToks[i] == "string" {
                    argType = String
                }
                // Append to the arguments array
                signature := state.funcs[name]
                signature.args = append(signature.args, argument{ argName, argType })
                state.funcs[name] = signature
            }

            // Skip to endfunc
            traverseTo("endfunc")

        case "endfunc":
            // Return to the call
            state.pc = pop(&state.functionCalls).returnLine
            if len(state.functionCalls) == 0 {
                state.executingFunction = false
            }

        case "return":
            // Return a value and stop function execution
            state.returnValue = evalExpr(lineToks[1:])
            state.pc = pop(&state.functionCalls).returnLine - 1
            if len(state.functionCalls) == 0 {
                state.executingFunction = false
            }

        case "print":
            if lineToks[1][0:1] == "\"" {
                // Found a string
                if string(lastChar(lineToks[1])) != "\"" {
                    log.Fatal("Unclosed string. You need to close the quotes or remove the spaces")
                }
                fmt.Println(stringContents(lineToks[1]))
            } else if val, exists := getWorkingScope().strings[lineToks[1]]; exists {
                fmt.Println(stringContents(val))
            } else if val, exists := state.globScope.strings[lineToks[1]]; exists {
                fmt.Println(stringContents(val))
            } else {
                // Found an expression to parse
                fmt.Println(evalExpr(lineToks[1:]))
            }

        // Dump our local scope's contents for debugging, or when we can't be bothered
        // remembering what variable we care about
        case "dump":
            //fmt.Println(getWorkingScope())
            fmt.Printf("Scope @pc=%d: %+v func?%t\n", state.pc, getWorkingScope(), state.executingFunction)

        case "input":
            if len(lineToks) <= 1 {
                log.Fatal("`input` called without destination variable")
            }
            varName := lineToks[1]

            // Read user input
            reader := bufio.NewReader(os.Stdin)
            fmt.Printf("%s=", varName)
            text, _ := reader.ReadString('\n')
            text = strings.Replace(text, "\n", "", -1)

            value, err := strconv.Atoi(text)
            if err != nil {
                log.Fatal("Input wasn't an int")
            }
            getWorkingScope().ints[varName] = value

        // Assume if it isn't a reserved word it's a variable or a function call
        default:
        // Function call
            if val, exists := state.funcs[lineToks[0]]; exists {
                if state.returnValue != 0 {
                    getWorkingScope().ints[lineToks[0]] = state.returnValue
                    state.returnValue = 0
                } else {
                    if len(lineToks) <= 1 ||
                        lineToks[1] != "(" || lineToks[len(lineToks) - 1] != ")" {
                        log.Fatal("Possible function call without parentheses")
                    }

                    state.functionCalls = append(state.functionCalls, functionCall{ state.pc, newScope() })
                    // We're not in a function yet so we can evaluate variable arguments,
                    // but we still need to assign argument VALUES to the variable scope
                    functionScope := last(&state.functionCalls).funcScope

                    state.parsingFunCallArgs = true
                    for i, arg := range val.args {
                        argExpr := make([]string, 0)
                        for j := i * 2 + 2; lineToks[j] != "," && lineToks[j] != ")"; j++ {
                            argExpr = append(argExpr, lineToks[j])
                        }

                        functionScope.ints[arg.argName] = evalExpr(argExpr)
                    }
                    state.parsingFunCallArgs = false

                    state.executingFunction = true
                    state.pc = val.startline
                }

        // Variable assignment
            } else if lineToks[1] == "=" {
                // Grab our scope: either global or function local
                workingScope := getWorkingScope()

                // Check if we're assigning a string
                if lineToks[2][0:1] == "\"" {
                    workingScope.strings[lineToks[0]] = lineToks[2]
                } else if val, exists := state.funcs[lineToks[2]]; exists {
                    if state.returnValue != 0 {
                        workingScope.ints[lineToks[0]] = state.returnValue
                        state.returnValue = 0
                    } else {
                        if len(lineToks) < 5 ||
                            lineToks[3] != "(" || last(&lineToks) != ")" {
                                log.Fatal("Possible function call without parentheses")
                        }

                        state.functionCalls = append(state.functionCalls, functionCall{ state.pc, newScope() })
                        functionScope := last(&state.functionCalls).funcScope

                        state.parsingFunCallArgs = true
                        for i, arg := range val.args {
                            argExpr := make([]string, 0)
                            for j := i*2 + 4; lineToks[j] != "," && lineToks[j] != ")"; j++ {
                                argExpr = append(argExpr, lineToks[j])
                            }

                            functionScope.ints[arg.argName] = evalExpr(argExpr)
                        }
                        state.parsingFunCallArgs = false
                        
                        state.executingFunction = true
                        state.pc = val.startline
                    }

                } else {
                    // In the variable map assign this var to the expression to the right of '='
                    newVal := evalExpr(lineToks[2:])
                    workingScope.ints[lineToks[0]] = newVal
                }
            } else {
                log.Fatal("Function or variable `", lineToks[0], "` doesn't exist")
            }

        }

        state.pc++
    }
}

// Evaluate a single arithmetic expression and return its value
func evalExpr(toks []string) int {
    stack := make([]int, 0)

    // Examine each token and deal with it
    for _, tok := range toks {
        // Handle integers
        if val, err := strconv.Atoi(tok); err == nil {
            stack = append(stack, val)
            continue
        }

        // Handle binary operators
        operators := []string{"+", "-", "*", "/", "%", ">", "<", ">=", "<=", "==", "!=", "&&", "||"}
        if slices.Contains(operators, tok) {
            if len(stack) < 2 {
                log.Fatal("Operator used before both elements are in the stack. Likely forgot to use Reverse Polish notation")
            }
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
            case "%":
                stack = append(stack, lhs % rhs)

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

            // Logical operators
            case "&&":
                stack = append(stack, Btoi(lhs!=0 && rhs!=0))
            case "||":
                stack = append(stack, Btoi(lhs!=0 || rhs!=0))
            }
            continue
        }

        // Handle variables
        // First, grab our scope
        workingScope := getWorkingScope()
        if state.parsingFunCallArgs && len(state.functionCalls) >= 2 {
            workingScope = state.functionCalls[len(state.functionCalls) - 2].funcScope
        }


        if val, exists := workingScope.ints[tok]; exists {
            stack = append(stack, val)
            continue
        } else {
            log.Fatalf("Variable `%s` does not exist", tok)
        }
    }

    return pop(&stack)
}

func main() {
    args := os.Args[1:]
    if len(args) < 1 {
        log.Fatal("No program given to execute")
    }
    fPath := args[0]

    dat, err := os.ReadFile(fPath)
    if err != nil {
        panic(err)
    }
    program := string(dat)

    toks := tokenizeProgram(program)

    eval(toks)

    // Dump the program state when we're done
    //fmt.Println(state.globScope)
}

