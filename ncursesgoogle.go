package main

// TODO: Separate ncurses window for menu (and another one for the search form?)
// TODO: Make CTRL+C safe
// TODO: Pagination of search results
// TODO: VIM-mode on the search form
// TODO: Tidy up and refactor the main function
// TODO: Colorize the google logo
// TODO: Help messages at the bottom
// TODO: Configuration file

/*
#cgo LDFLAGS: -lncursesw -lformw
#include <stdlib.h>
#include <locale.h>
#include <ncurses.h>
#include <form.h>
#include "ioctl_wrapper.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"html"
	"os"
	"os/exec"
	"strings"
)

var art []string = []string{
	"  ___                _",
	" / __|___  ___  __ _| |___ ",
	"| (_ / _ \\/ _ \\/ _` |   -_)",
	" \\___\\___/\\___/\\__, |_\\___|",
	"               |___/",
}

var colors map[int]map[int]int

var (
	browser_cmd  string = "aurora"
	browser_args string = "-new-window"
)

type MenuItem struct {
	Text string
	Url  string
}

const TIOCGWINSZ C.ulong = 0x5413

func GetTerminalSize() (int, int) {
	var ts C.ttysize_t
	C.myioctl(0, TIOCGWINSZ, &ts)

	return int(ts.ws_col), int(ts.ws_row)
}

func draw_menu(x, y int, items []MenuItem, highlight int) {
	for i, item := range items {
		if i == highlight {
			C.attron(C.A_REVERSE)
			C.mvaddstr(C.int(y), C.int(x), C.CString(item.Text))
			C.attroff(C.A_REVERSE)
		} else {
			C.mvaddstr(C.int(y), C.int(x), C.CString(item.Text))
		}
		y += 1
	}
}

func set_color(x, y, color int) {
	if m, ok := colors[x]; ok {
		m[y] = color
	} else {
		colors[x] = make(map[int]int)
		colors[x][y] = color
	}
}

func get_color(x, y int) (color int, err error) {
	if _, ok := colors[x]; ok {
		if color, ok = colors[x][y]; !ok {
			err = errors.New("color for position not found")
		}
	} else {
		err = errors.New("color for position not found")
	}

	return
}

func setup_colors() {
	C.init_pair(1, C.COLOR_BLUE, C.COLOR_BLACK)
	C.init_pair(2, C.COLOR_RED, C.COLOR_BLACK)
	C.init_pair(3, C.COLOR_YELLOW, C.COLOR_BLACK)
	C.init_pair(4, C.COLOR_GREEN, C.COLOR_BLACK)

	colors = make(map[int]map[int]int)

	// G
	set_color(0, 0, 1)
	set_color(0, 1, 1)
	set_color(0, 2, 1)
	set_color(0, 3, 1)

	// o
	set_color(5, 0, 2)
	set_color(6, 1, 2)
	set_color(5, 2, 2)
	set_color(5, 3, 2)

	// o
	set_color(10, 0, 3)
	set_color(10, 1, 3)
	set_color(10, 2, 3)
	set_color(10, 3, 3)

	// g
	set_color(15, 0, 1)
	set_color(15, 1, 1)
	set_color(15, 2, 1)
	set_color(15, 3, 1)
	set_color(15, 4, 1)

	// l
	set_color(20, 0, 4)
	set_color(20, 1, 4)
	set_color(20, 2, 4)
	set_color(20, 3, 4)

	// e
	set_color(23, 0, 2)
	set_color(23, 1, 2)
	set_color(23, 2, 2)
	set_color(22, 3, 2)
}

func draw_logo(xpos, ypos int) {
	current_color := -1
	for i, line := range art {
		for j, char := range line {
			if color, err := get_color(j, i); err == nil {
				current_color = color
				C.attron(C.COLOR_PAIR(C.int(color)))
			}

			C.mvaddch(C.int(ypos + i), C.int(xpos + j), C.chtype(char))
		}
	}

	if current_color != -1 {
		C.attroff(C.COLOR_PAIR(C.int(current_color)))
	}

}

func doQuery(query string) (response Response, err error) {
	response, err = Google(query)

	if err != nil {
		return
	}

	if response.ResponseData == nil || response.ResponseData.Results == nil {
		err = errors.New(
			fmt.Sprintf("Invalid response on google query: %s", response))
		return
	}

	return
}

func getMenuListFromQuery(response Response) (menu_list []MenuItem) {
	results := *response.ResponseData.Results

	for _, result := range results {
		item := MenuItem{html.UnescapeString(result.TitleNoFormatting), result.Url}
		menu_list = append(menu_list, item)
	}

	return
}

func main() {
	var err error
	var query string = ""
	var response Response
	var menu_list []MenuItem
	var isEditMode bool = false
	highlight := 0
	width, _ := GetTerminalSize()

	xpos := (width / 2.0) - (80 / 2.0)
	ypos := 20

	if len(os.Args) > 1 {
		query = strings.Join(os.Args[1:], " ")
		if response, err = Google(query); err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			return
		}

		menu_list = getMenuListFromQuery(response)
	}

	// Enable unicode in ncurses
	C.setlocale(C.LC_ALL, C.CString(""))

	C.initscr()
	C.curs_set(0)
	C.noecho()
	C.keypad(C.stdscr, true)
	C.cbreak()

	C.start_color()
	setup_colors()

	var fields [2]*C.FIELD
	fields[0] = C.new_field(1, 80, 17, C.int(xpos), 0, 0)
	fields[1] = nil

	var search_form *C.FORM = C.new_form(&fields[0])
	C.post_form(search_form)

	draw_logo(xpos, 10)

	if len(os.Args) <= 1 {
		isEditMode = true
		C.curs_set(1)
	} else {
		C.set_field_buffer(fields[0], 0, C.CString(query))
		draw_menu(xpos, ypos, menu_list, highlight)
	}

	C.form_driver(search_form, C.REQ_END_LINE)
	C.refresh()

ui_loop:
	for {
		if isEditMode {
			c := C.getch()
			switch c {
			case C.KEY_LEFT:
				C.form_driver(search_form, C.REQ_PREV_CHAR)
			case C.KEY_RIGHT:
				C.form_driver(search_form, C.REQ_NEXT_CHAR)
			case 27: // Esc
				err = nil
				break ui_loop
			case '\n':
				C.form_driver(search_form, C.REQ_END_LINE)
				C.form_driver(search_form, C.REQ_VALIDATION)

				// 0 is the display buffer
				str := C.field_buffer(fields[0], 0)
				isEditMode = false
				C.curs_set(0)

				if response, err = doQuery(C.GoString(str)); err != nil {
					break ui_loop
				}

				menu_list = getMenuListFromQuery(response)

				// Clear old results from the display
				C.clrtobot()
				C.refresh()
				highlight = 0
			case '\t':
				isEditMode = false
				C.curs_set(0)
			case C.KEY_BACKSPACE:
				C.form_driver(search_form, C.REQ_PREV_CHAR)
				C.form_driver(search_form, C.REQ_DEL_CHAR)
			case 127: // Backspace
				C.form_driver(search_form, C.REQ_PREV_CHAR)
				C.form_driver(search_form, C.REQ_DEL_CHAR)
			default:
				C.form_driver(search_form, c)
			}
		} else {
			draw_menu(xpos, ypos, menu_list, highlight)
			c := C.getch()

			switch c {
			case 27: // Esc
				err = nil
				break ui_loop
			case '\t':
				C.form_driver(search_form, C.REQ_END_LINE)
				C.curs_set(1)
				isEditMode = true
			case C.KEY_DOWN:
				if highlight < len(menu_list)-1 {
					highlight += 1
				}
			case 'j':
				if highlight < len(menu_list)-1 {
					highlight += 1
				}
			case C.KEY_UP:
				if highlight > 0 {
					highlight -= 1
				}
			case 'k':
				if highlight > 0 {
					highlight -= 1
				}
			case 'q':
				err = nil
				break ui_loop
			case '\n':
				cmd := exec.Command(
					browser_cmd, browser_args, menu_list[highlight].Url)
				err = cmd.Start()
				break ui_loop
			default:
				continue
			}
		}

		C.refresh()
	}

	C.unpost_form(search_form)
	C.free_form(search_form)
	C.free_field(fields[0])

	C.echo()
	C.nocbreak()
	C.keypad(C.stdscr, false)

	C.endwin()

	// Print any errors if we didn't exit normally
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
	}
}
