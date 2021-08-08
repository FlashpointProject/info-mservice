package main

type DbResultError struct {
	status  int
	message string
	err     error
}
