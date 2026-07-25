package dice

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var cmdGen = &CmdItemInfo{
	Name:      "gen",
	ShortHelp: ".gen {roll|rollx} [数量]",
	Help: "PMDnD 角色属性生成:\n" +
		".gen roll [数量]     生成一组属性（6个数值，自由分配）\n" +
		".gen rollx [数量]    生成一组属性（带属性名，预设模式）\n" +
		"默认生成1组，最多10组",
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		sub := cmdArgs.GetArgN(1)
		isMode2 := sub == "rollx" || cmdArgs.Command == "pmdndx"
		if sub == "roll" || sub == "rollx" {
			// 移除第一个参数（子命令）
			if len(cmdArgs.Args) > 0 {
				cmdArgs.Args = cmdArgs.Args[1:]
			}
		}
		n := cmdArgs.GetArgN(1)
		val, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			if n == "" {
				val = 1
			} else {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
		}
		if val > 10 {
			val = 10
		}
		var ss []string
		for i := int64(0); i < val; i++ {
			if isMode2 {
				r := ctx.EvalFString(`力量:{$t1=4d6k3} 体质:{$t2=4d6k3} 敏捷:{$t3=4d6k3} 智力:{$t4=4d6k3} 感知:{$t5=4d6k3} 魅力:{$t6=4d6k3} 共计:{$tT=$t1+$t2+$t3+$t4+$t5+$t6}`, nil)
				if r.vm.Error != nil {
					break
				}
				result := r.ToString() + "\n"
				ss = append(ss, result)
			} else {
				r := ctx.EvalFString(`{4d6k3}, {4d6k3}, {4d6k3}, {4d6k3}, {4d6k3}, {4d6k3}`, nil)
				if r.vm.Error != nil {
					break
				}
				result := r.ToString()
				var nums Int64SliceDesc
				total := int64(0)
				for _, i := range strings.Split(result, ", ") {
					val, _ := strconv.ParseInt(i, 10, 64)
					nums = append(nums, val)
					total += val
				}
				sort.Sort(nums)
				var items []string
				for _, i := range nums {
					items = append(items, strconv.FormatInt(i, 10))
				}
				ret := fmt.Sprintf("[%s] = %d\n", strings.Join(items, ", "), total)
				ss = append(ss, ret)
			}
		}
		sep := DiceFormatTmpl(ctx, "DND:制卡_分隔符")
		info := strings.Join(ss, sep)
		VarSetValueStr(ctx, "$t制卡结果文本", info)
		var text string
		if isMode2 {
			text = DiceFormatTmpl(ctx, "DND:制卡_预设模式")
		} else {
			text = DiceFormatTmpl(ctx, "DND:制卡_自由分配模式")
		}
		ReplyToSender(ctx, msg, text)
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
