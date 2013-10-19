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

	var fields [2]*C.FIELD
	fields[0] = C.new_field(1, 80, 17, C.int(xpos), 0, 0)
	fields[1] = nil

	var search_form *C.FORM = C.new_form(&fields[0])
	C.post_form(search_form)

	y := 10
	for _, item := range art {
		C.mvaddstr(C.int(y), C.int(xpos), C.CString(item))
		y += 1
	}

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
			case 'j':
				if highlight < len(menu_list)-1 {
					highlight += 1
				}
			case 'k':
				if highlight > 0 {
					highlight -= 1
				}
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

	C.endwin()

	// Print any errors if we didn't exit normally
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
	}
}
