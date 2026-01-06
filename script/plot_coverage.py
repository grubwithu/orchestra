#!/usr/bin/env python3
import sys
import re
import matplotlib.pyplot as plt
from datetime import datetime

def parse_log_file(file_path):
    """
    解析日志文件，提取时间和覆盖率数据
    """
    time_coverage_pairs = []
    pattern = r'(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) Covearge Update:\s+(\d+)'
    
    with open(file_path, 'r') as f:
        for line in f:
            match = re.search(pattern, line)
            if match:
                time_str = match.group(1)
                coverage = int(match.group(2))
                time_obj = datetime.strptime(time_str, '%Y/%m/%d %H:%M:%S')
                time_coverage_pairs.append((time_obj, coverage))
    
    return time_coverage_pairs

def plot_coverage(time_coverage_pairs):
    """
    绘制覆盖率曲线图
    """
    if not time_coverage_pairs:
        print("没有找到有效的覆盖率数据")
        return
    
    # 提取时间和覆盖率
    times = [pair[0] for pair in time_coverage_pairs]
    coverages = [pair[1] for pair in time_coverage_pairs]
    
    # 计算相对时间（以第一个数据点为0）
    base_time = times[0]
    relative_times = [(t - base_time).total_seconds() / 60 for t in times]
    
    # 创建图表
    plt.figure(figsize=(10, 6))
    plt.plot(relative_times, coverages, linestyle='-')
    plt.title('Coverage Over Time')
    plt.xlabel('Time (minutes from start)')
    plt.ylabel('Coverage')
    plt.grid(True)
    
    # 保存图表
    plt.savefig('coverage_plot.png')
    print("图表已保存为 coverage_plot.png")
    plt.show()

def main():
    if len(sys.argv) != 2:
        print("用法: python plot_coverage.py <log_file>")
        sys.exit(1)
    
    log_file = sys.argv[1]
    try:
        time_coverage_pairs = parse_log_file(log_file)
        plot_coverage(time_coverage_pairs)
    except FileNotFoundError:
        print(f"错误: 找不到文件 {log_file}")
        sys.exit(1)
    except Exception as e:
        print(f"错误: {str(e)}")
        sys.exit(1)

if __name__ == "__main__":
    main()