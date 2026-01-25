#!/usr/bin/env python3
import sys
import re

def main():
    # 检查命令行参数
    if len(sys.argv) != 2:
        print("Usage: python3 parse_log.py <log_file_path>")
        sys.exit(1)
    
    log_file_path = sys.argv[1]
    
    total_increases = 0
    important_increases = 0
    try:
        # 打开并读取日志文件
        with open(log_file_path, 'r', encoding='utf-8') as f:
            # 定义正则表达式模式
            # 匹配 "Fuzzer <name> find <num1> increases in total, <num2> of them are important"
            pattern = r'Fuzzer\s+(\w+)\s+find\s+(\d+)\s+increases\s+in\s+total,\s+(\d+)\s+of\s+them\s+are\s+important'
            
            # 逐行处理日志
            for line_num, line in enumerate(f, 1):
                line = line.strip()
                if not line:
                    continue
                
                # 使用正则表达式匹配
                match = re.search(pattern, line)
                if match:
                    # 提取信息
                    fuzzer_name = match.group(1)
                    total_increases += int(match.group(2))
                    important_increases += int(match.group(3))
                    
                    # 打印提取的信息（可以根据需要进行后续处理）
                    print(f"Line {line_num}: Fuzzer={fuzzer_name}, Total={total_increases}, Important={important_increases}")
                else:
                    # 可选：处理不匹配的行
                    print(f"Line {line_num}: No match found")
                    
    except FileNotFoundError:
        print(f"Error: File '{log_file_path}' not found.")
        sys.exit(1)
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)

    print(f"Total increases: {total_increases}")
    print(f"Important increases: {important_increases}")
    print(f"Percent: {important_increases / total_increases * 100:.2f}%")

if __name__ == "__main__":
    main()