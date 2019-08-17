package readline

import (
	"bytes"
	"strings"
)

// Caller type for dynamic completion
type DynamicCompleteFunc func(string) []string

type PrefixCompleterInterface interface {
	Print(prefix string, level int, buf *bytes.Buffer)
	Do(line []rune, pos int) (newLine [][]rune, length int)
	GetName() []rune
	GetChildren() []PrefixCompleterInterface
	SetChildren(children []PrefixCompleterInterface)
}

type DynamicPrefixCompleterInterface interface {
	PrefixCompleterInterface
	IsDynamic() bool
	GetDynamicNames(line []rune) [][]rune
}

type PrefixCompleter struct {
	Name     []rune
	Dynamic  bool
	Callback DynamicCompleteFunc
	Children []PrefixCompleterInterface
}

func (p *PrefixCompleter) Tree(prefix string) string {
	buf := bytes.NewBuffer(nil)
	p.Print(prefix, 0, buf)
	return buf.String()
}

func Print(p PrefixCompleterInterface, prefix string, level int, buf *bytes.Buffer) {
	if strings.TrimSpace(string(p.GetName())) != "" {
		buf.WriteString(prefix)
		if level > 0 {
			buf.WriteString("├")
			buf.WriteString(strings.Repeat("─", (level*4)-2))
			buf.WriteString(" ")
		}
		buf.WriteString(string(p.GetName()) + "\n")
		level++
	}
	for _, ch := range p.GetChildren() {
		ch.Print(prefix, level, buf)
	}
}

func (p *PrefixCompleter) Print(prefix string, level int, buf *bytes.Buffer) {
	Print(p, prefix, level, buf)
}

func (p *PrefixCompleter) IsDynamic() bool {
	return p.Dynamic
}

func (p *PrefixCompleter) GetName() []rune {
	return p.Name
}

func (p *PrefixCompleter) GetDynamicNames(line []rune) [][]rune {
	var names = [][]rune{}
	for _, name := range p.Callback(string(line)) {
		names = append(names, []rune(name+" "))
	}
	return names
}

func (p *PrefixCompleter) GetChildren() []PrefixCompleterInterface {
	return p.Children
}

func (p *PrefixCompleter) SetChildren(children []PrefixCompleterInterface) {
	p.Children = children
}

func NewPrefixCompleter(pc ...PrefixCompleterInterface) *PrefixCompleter {
	return PcItem("", pc...)
}

func PcItem(name string, pc ...PrefixCompleterInterface) *PrefixCompleter {
	name += " "
	return &PrefixCompleter{
		Name:     []rune(name),
		Dynamic:  false,
		Children: pc,
	}
}

func PcItemDynamic(callback DynamicCompleteFunc, pc ...PrefixCompleterInterface) *PrefixCompleter {
	return &PrefixCompleter{
		Callback: callback,
		Dynamic:  true,
		Children: pc,
	}
}

func (p *PrefixCompleter) Do(line []rune, pos int) (newLine [][]rune, offset int) {
	//return doInternal(p, line, pos, line)
	newLine, offset = doInternal(p, line, pos, line)
	return doInternal2(p, line, offset, newLine)
}

func Do(p PrefixCompleterInterface, line []rune, pos int) (newLine [][]rune, offset int) {
	//return doInternal2(p, line, offset, newLine)
	newLine, offset = doInternal(p, line, pos, line)
	return doInternal2(p, line, offset, newLine)
}

func doInternal(p PrefixCompleterInterface, line []rune, pos int, origLine []rune) (newLine [][]rune, offset int) {
	line = runes.TrimSpaceLeft(line[:pos])
	goNext := false
	var lineCompleter PrefixCompleterInterface
	for _, child := range p.GetChildren() {
		childNames := make([][]rune, 1)

		childDynamic, ok := child.(DynamicPrefixCompleterInterface)
		if ok && childDynamic.IsDynamic() {
			continue
			//childNames = childDynamic.GetDynamicNames(line)
		} else {
			childNames[0] = child.GetName()
		}
		for _, childName := range childNames {
			if len(line) >= len(childName) {
				if runes.HasPrefix(line, childName) {
					if len(line) == len(childName) {
						newLine = append(newLine, []rune{' '})
					} else {
						newLine = append(newLine, childName)
					}
					offset = len(childName)
					lineCompleter = child
					goNext = true
				}
			} else {
				if runes.HasPrefix(childName, line) {
					newLine = append(newLine, childName[len(line):])
					offset = len(line)
					lineCompleter = child
				}
			}
		}
	}

	if len(newLine) != 1 {
		return
	}

	tmpLine := make([]rune, 0, len(line))
	for i := offset; i < len(line); i++ {
		if line[i] == ' ' {
			continue
		}

		tmpLine = append(tmpLine, line[i:]...)
		return doInternal(lineCompleter, tmpLine, len(tmpLine), origLine)
	}

	if goNext {
		return doInternal(lineCompleter, nil, 0, origLine)
	}
	return
}

func doInternal2(p PrefixCompleterInterface, line []rune, offset_in int, newLine_in [][]rune) (newLine [][]rune, offset int) {
	offset = offset_in
	newLine = newLine_in

	if len(newLine) > 0 && len(newLine[0]) == 5 &&
		newLine[0][0] == 's' && newLine[0][1] == 'e' && newLine[0][2] == 'n' && newLine[0][3] == 'd' && newLine[0][4] == 32 {
		return
	}

	if len(line) == 0 || line[len(line)-1] == ' ' {
		line = []rune("")
	} else {
		for i := len(line) - 1; i >= 0; i-- {
			if line[i] == ' ' {
				line = line[i+1:]
				break
			}
		}
	}

	for _, child := range p.GetChildren() {
		childNames := make([][]rune, 1)
		childDynamic, ok := child.(DynamicPrefixCompleterInterface)
		if ok && childDynamic.IsDynamic() {
			childNames = childDynamic.GetDynamicNames(getDir(line))
			line = getName(line)
		} else {
			continue
		}

		for _, childName := range childNames {
			if len(line) >= len(childName) {
				if runes.HasPrefix(line, childName) {
					if len(line) == len(childName) {
						newLine = append(newLine, []rune{' '})
					} else {
						newLine = append(newLine, childName)
					}
					offset = len(childName)
				}
			} else {

				if runes.HasPrefix(childName, line) {
					newLine = append(newLine, childName[len(line):])
					offset = len(line)
				}
			}
		}
	}
	return
}

func getDir(in []rune) (line []rune) {
	if len(in) == 0 || in[len(in)-1] == '/' || in[len(in)-1] == '\\' {
		return in
	} else {
		for i := len(in) - 1; i >= 0; i-- {
			if in[i] == '/' || in[i] == '\\' {
				line = in[:i+1]
				return
			}
		}
	}
	return []rune{}
}

func getName(in []rune) (line []rune) {
	if len(in) == 0 || in[len(in)-1] == '/' || in[len(in)-1] == '\\' {
		line = []rune("")
		return
	} else {
		for i := len(in) - 1; i >= 0; i-- {
			if in[i] == '/' || in[i] == '\\' {
				line = in[i+1:]
				return
			}
		}
	}
	return in
}
