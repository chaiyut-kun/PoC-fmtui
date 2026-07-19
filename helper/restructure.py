import os

def restructure():
    # Directories
    tui_dir = 'internal/tui'
    
    # Read model.go
    with open('model.go', 'r') as f:
        model_lines = f.readlines()
    
    # Find boundary indices
    update_start = -1
    view_start = -1
    for i, line in enumerate(model_lines):
        if line.startswith('func (m model) Update('):
            update_start = i
        elif line.startswith('func (m model) View('):
            view_start = i
            
    if update_start == -1 or view_start == -1:
        print("Error: Could not find Update or View functions in model.go")
        return
        
    print(f"Update starts at line {update_start + 1}, View starts at line {view_start + 1}")
    
    # Header/imports logic (lines 1 to 15 are package and imports)
    # We will reuse imports for each file.
    # For simplicity, let's build the package header for each file:
    package_header = "package tui\n\n"
    
    # Write internal/tui/model.go
    model_content = package_header
    # Copy imports + helper functions and structs (excluding Update and View)
    for i in range(2, update_start):
        line = model_lines[i]
        # Change return type of InitialModel() to tea.Model
        if line.startswith('func InitialModel() model'):
            line = 'func InitialModel() tea.Model {\n'
        model_content += line
        
    with open(os.path.join(tui_dir, 'model.go'), 'w') as f:
        f.write(model_content)
        
    # Write internal/tui/update.go
    update_content = package_header + "import (\n\ttea \"charm.land/bubbletea/v2\"\n\t\"os\"\n\t\"os/exec\"\n\t\"strings\"\n)\n\n"
    for i in range(update_start, view_start):
        update_content += model_lines[i]
        
    with open(os.path.join(tui_dir, 'update.go'), 'w') as f:
        f.write(update_content)
        
    # Write internal/tui/view.go
    view_content = package_header + "import (\n\t\"fmt\"\n\t\"strings\"\n\ttea \"charm.land/bubbletea/v2\"\n\t\"github.com/charmbracelet/lipgloss\"\n)\n\n"
    for i in range(view_start, len(model_lines)):
        view_content += model_lines[i]
        
    with open(os.path.join(tui_dir, 'view.go'), 'w') as f:
        f.write(view_content)
        
    # Helper to rewrite package header in other Go files
    def rewrite_package(filename, new_path):
        with open(filename, 'r') as f:
            lines = f.readlines()
        
        # Replace first line (package main) with package tui
        if lines and lines[0].startswith('package main'):
            lines[0] = 'package tui\n'
            
        with open(new_path, 'w') as f:
            f.writelines(lines)
            
    rewrite_package('bookmarks.go', os.path.join(tui_dir, 'bookmarks.go'))
    rewrite_package('constant.go', os.path.join(tui_dir, 'constant.go'))
    rewrite_package('utilities.go', os.path.join(tui_dir, 'utilities.go'))
    rewrite_package('main_test.go', os.path.join(tui_dir, 'tui_test.go'))
    
    print("Files successfully split and moved to internal/tui/")

if __name__ == '__main__':
    restructure()
