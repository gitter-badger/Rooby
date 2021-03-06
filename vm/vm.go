package vm

import (
	"bytes"
	"fmt"
	"strings"
)

type VM struct {
	CallFrameStack *CallFrameStack
	Stack          *Stack
	SP             int
	CFP            int
	Constants      map[string]*Pointer
	LabelTable     map[LabelType]map[string][]*InstructionSet
	MethodISTable  *ISIndexTable
	ClassISTable   *ISIndexTable
	BlockList      *ISIndexTable
}

type ISIndexTable struct {
	Data map[string]int
}

type Stack struct {
	Data []*Pointer
	VM   *VM
}

func New() *VM {
	s := &Stack{}
	cfs := &CallFrameStack{CallFrames: []*CallFrame{}}
	vm := &VM{Stack: s, CallFrameStack: cfs, SP: 0, CFP: 0}
	s.VM = vm
	cfs.VM = vm

	vm.initConstants()
	vm.MethodISTable = &ISIndexTable{Data: make(map[string]int)}
	vm.ClassISTable = &ISIndexTable{Data: make(map[string]int)}
	vm.BlockList = &ISIndexTable{Data: make(map[string]int)}
	vm.LabelTable = map[LabelType]map[string][]*InstructionSet{
		LABEL_DEF:      make(map[string][]*InstructionSet),
		LABEL_DEFCLASS: make(map[string][]*InstructionSet),
		BLOCK:          make(map[string][]*InstructionSet),
		PROGRAM:        make(map[string][]*InstructionSet),
	}

	return vm
}

func (vm *VM) EvalCallFrame(cf *CallFrame) {
	for cf.PC < len(cf.InstructionSet.Instructions) {
		i := cf.InstructionSet.Instructions[cf.PC]
		vm.execInstruction(cf, i)
	}
}

func (vm *VM) Exec() {
	cf := vm.CallFrameStack.Top()
	vm.EvalCallFrame(cf)
}

func (vm *VM) initConstants() {
	constants := make(map[string]*Pointer)

	builtInClasses := []Class{
		IntegerClass,
		StringClass,
		BooleanClass,
		NullClass,
		ArrayClass,
		HashClass,
		ClassClass,
		ObjectClass,
	}

	for _, c := range builtInClasses {
		p := &Pointer{Target: c}
		constants[c.ReturnName()] = p
	}

	vm.Constants = constants
}

func (vm *VM) execInstruction(cf *CallFrame, i *Instruction) {
	cf.PC += 1
	//fmt.Print(i.Inspect())
	i.Action.Operation(vm, cf, i.Params...)
	//fmt.Println(vm.CallFrameStack.inspect())
	//fmt.Println(vm.Stack.inspect())
}

func (vm *VM) getBlock(name string) (*InstructionSet, bool) {
	// The "name" here is actually an index from label
	// for example <Block:1>'s name is "1"
	iss, ok := vm.LabelTable[BLOCK][name]

	if !ok {
		return nil, false
	}

	is := iss[0]

	return is, ok
}

func (vm *VM) getMethodIS(name string) (*InstructionSet, bool) {
	iss, ok := vm.LabelTable[LABEL_DEF][name]

	if !ok {
		return nil, false
	}

	is := iss[vm.MethodISTable.Data[name]]

	vm.MethodISTable.Data[name] += 1
	return is, ok
}

func (vm *VM) getClassIS(name string) (*InstructionSet, bool) {
	iss, ok := vm.LabelTable[LABEL_DEFCLASS][name]

	if !ok {
		return nil, false
	}

	is := iss[vm.ClassISTable.Data[name]]

	vm.ClassISTable.Data[name] += 1
	return is, ok
}

func (vm *VM) setLabel(is *InstructionSet, name string) {
	var l *Label
	var labelName string
	var labelType LabelType

	if name == "ProgramStart" {
		labelName = name
		labelType = PROGRAM

	} else {
		labelName = strings.Split(name, ":")[1]
		labelType = labelTypes[strings.Split(name, ":")[0]]
	}

	l = &Label{Name: name, Type: labelType}
	is.Label = l
	vm.LabelTable[labelType][labelName] = append(vm.LabelTable[labelType][labelName], is)
}

func (s *Stack) push(v *Pointer) {
	if len(s.Data) <= s.VM.SP {
		s.Data = append(s.Data, v)
	} else {
		s.Data[s.VM.SP] = v
	}

	s.VM.SP += 1
}

func (s *Stack) pop() *Pointer {
	if len(s.Data) < 1 {
		panic("Nothing to pop!")
	}

	s.VM.SP -= 1

	v := s.Data[s.VM.SP]
	s.Data[s.VM.SP] = nil
	return v
}

func (s *Stack) Top() *Pointer {

	if s.VM.SP > 0 {
		return s.Data[s.VM.SP-1]
	}

	return s.Data[0]
}

func (s *Stack) inspect() string {
	var out bytes.Buffer
	datas := []string{}

	for i, p := range s.Data {
		if p != nil {
			o := p.Target
			if i == s.VM.SP {
				datas = append(datas, fmt.Sprintf("%s (%T) %d <----", o.Inspect(), o, i))
			} else {
				datas = append(datas, fmt.Sprintf("%s (%T) %d", o.Inspect(), o, i))
			}

		} else {
			if i == s.VM.SP {
				datas = append(datas, "nil <----")
			} else {
				datas = append(datas, "nil")
			}

		}

	}

	out.WriteString("-----------\n")
	out.WriteString(strings.Join(datas, "\n"))
	out.WriteString("\n---------\n")

	return out.String()
}

func newError(format string, args ...interface{}) *Error {
	return &Error{Message: fmt.Sprintf(format, args...)}
}
