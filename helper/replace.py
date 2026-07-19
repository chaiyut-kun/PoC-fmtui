import sys

with open('model.go', 'r') as f:
    lines = f.readlines()

new_lines = []
for i, line in enumerate(lines):
    # Do not replace inside the visibleFiles function itself
    if 98 <= i <= 104:
        new_lines.append(line)
        continue
    
    new_line = line.replace('t.filesPath', 't.visibleFiles()')
    new_lines.append(new_line)

with open('model.go', 'w') as f:
    f.writelines(new_lines)
