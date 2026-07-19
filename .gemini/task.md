# Checklist: Asynchronous Copy Content Feature

- [x] Define new message structures and add fields to the `model` struct in `model.go`
- [x] Implement asynchronous chunk-by-chunk copy goroutine and progress channel listener in `model.go`
- [x] Add case option "c" inside `copyMenuOpen` block to start copy content job
- [x] Handle `copyProgressMsg` and `copyFinishedMsg` inside `Update()` loop
- [x] Implement copy cancellation when `esc` is pressed and `copyJobActive` is true
- [x] Modify height allocation and render progress bar at the bottom of the File Explorer
- [x] Write unit tests for copy content, binary warning, and cancellation in `main_test.go`
- [x] Verify everything compiles and passes tests
