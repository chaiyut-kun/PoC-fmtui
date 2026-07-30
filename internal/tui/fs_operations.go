package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err == nil {
		_ = os.Chmod(dst, info.Mode())
	}
	return nil
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFileOrDir(src, dst string) error {
	absSrc, err := filepath.Abs(src)
	if err == nil {
		absDst, err := filepath.Abs(dst)
		if err == nil && absSrc == absDst {
			return fmt.Errorf("cannot copy file/directory to itself")
		}
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func moveFileOrDir(src, dst string) error {
	// Try atomic rename first
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	info, statErr := os.Stat(src)
	if statErr != nil {
		return statErr
	}

	if info.IsDir() {
		if copyErr := copyDir(src, dst); copyErr != nil {
			return copyErr
		}
		return os.RemoveAll(src)
	}

	if copyErr := copyFile(src, dst); copyErr != nil {
		return copyErr
	}
	return os.Remove(src)
}

func permanentDelete(path string) error {
	return os.RemoveAll(path)
}

func normalDelete(path string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	trashDir := filepath.Join(homeDir, ".local", "share", "Trash", "files")
	if err := os.MkdirAll(trashDir, 0755); err != nil {
		return "", err
	}

	baseName := filepath.Base(path)
	destination := filepath.Join(trashDir, baseName)
	counter := 1
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	for {
		if _, statErr := os.Stat(destination); os.IsNotExist(statErr) {
			break
		}
		destination = filepath.Join(trashDir, fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext))
		counter++
	}

	if err := moveFileOrDir(path, destination); err != nil {
		return "", err
	}

	return destination, nil
}

// ponytail: stdlib os.Stat & os.Chmod to toggle executable mode directly
func toggleExecutable(path string) (oldMode, newMode os.FileMode, err error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0, err
	}
	oldMode = info.Mode()
	if oldMode&0111 != 0 {
		newMode = oldMode &^ 0111
	} else {
		newMode = oldMode | 0111
	}
	if err := os.Chmod(path, newMode); err != nil {
		return 0, 0, err
	}
	return oldMode, newMode, nil
}

func (m *model) pushAction(action ActionRecord) {
	m.undoStack = append(m.undoStack, action)
	m.redoStack = nil
}

func (m *model) performUndo() error {
	if len(m.undoStack) == 0 {
		return fmt.Errorf("nothing to undo")
	}

	lastIdx := len(m.undoStack) - 1
	action := m.undoStack[lastIdx]
	m.undoStack = m.undoStack[:lastIdx]

	switch action.Type {
	case ActionDelete:
		if action.TrashPath == "" {
			return fmt.Errorf("cannot undo permanent delete")
		}
		if err := moveFileOrDir(action.TrashPath, action.OriginalPath); err != nil {
			return err
		}
	case ActionCreate:
		trashPath, err := normalDelete(action.OriginalPath)
		if err != nil {
			return err
		}
		action.TrashPath = trashPath
	case ActionRename:
		if err := moveFileOrDir(action.NewPath, action.OldPath); err != nil {
			return err
		}
	case ActionChmod:
		if err := os.Chmod(action.OriginalPath, action.OldMode); err != nil {
			return err
		}
	}

	m.redoStack = append(m.redoStack, action)
	m.reloadCurrentDir()
	return nil
}

func (m *model) performRedo() error {
	if len(m.redoStack) == 0 {
		return fmt.Errorf("nothing to redo")
	}

	lastIdx := len(m.redoStack) - 1
	action := m.redoStack[lastIdx]
	m.redoStack = m.redoStack[:lastIdx]

	switch action.Type {
	case ActionDelete:
		if err := moveFileOrDir(action.OriginalPath, action.TrashPath); err != nil {
			return err
		}
	case ActionCreate:
		if action.TrashPath != "" {
			if err := moveFileOrDir(action.TrashPath, action.OriginalPath); err != nil {
				return err
			}
		} else {
			if action.IsDir {
				if err := os.MkdirAll(action.OriginalPath, 0755); err != nil {
					return err
				}
			} else {
				file, err := os.Create(action.OriginalPath)
				if err != nil {
					return err
				}
				file.Close()
			}
		}
	case ActionRename:
		if err := moveFileOrDir(action.OldPath, action.NewPath); err != nil {
			return err
		}
	case ActionChmod:
		if err := os.Chmod(action.OriginalPath, action.NewMode); err != nil {
			return err
		}
	}

	m.undoStack = append(m.undoStack, action)
	m.reloadCurrentDir()
	return nil
}
