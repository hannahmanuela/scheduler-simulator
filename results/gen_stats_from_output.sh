#!/bin/bash


input_file="results/out.txt"

# get number of times each type of proc was late
# countStatic=$(grep -c 'done.*static' "$input_file")
# countDynamic=$(grep -c 'done.*dynamic' "$input_file")
# countFg=$(grep -c 'done.*fg' "$input_file")
# countBg=$(grep -c 'done.*bg' "$input_file")

# echo "Number of late static procs: $countStatic"
# echo "Number of late dynamic procs: $countDynamic"
# echo "Number of late fg procs: $countFg"
# echo "Number of late bg procs: $countBg"

# put proc done and proc created stats into different files
output_file1="results/procs_done.txt"
output_file2="results/procs_added.txt"
output_file3="results/procs_current.txt"
output_file4="results/procs_killed.txt"
output_file5="results/sched.txt"

# clear files first
>  "$output_file1"
>  "$output_file2"
>  "$output_file3"
>  "$output_file4"
>  "$output_file5"

# scrape output and separate data
while IFS= read -r line; do
    if [[ $line == *"done: "* ]]; then
        numbers=$(echo "$line" | sed -n -e 's/done: //p')
        echo "$numbers" >> "$output_file1"
    fi
    if [[ $line == *"adding: "* ]]; then
        numbers=$(echo "$line" | sed -n -e 's/adding: //p')
        echo "$numbers" >> "$output_file2"
    fi
    if [[ $line == *"current: "* ]]; then
        numbers=$(echo "$line" | sed -n -e 's/current: //p')
        echo "$numbers" >> "$output_file3"
    fi
    if [[ $line == *"killing: "* ]]; then
        numbers=$(echo "$line" | sed -n -e 's/killing: //p')
        echo "$numbers" >> "$output_file4"
    fi
    if [[ $line == *"sched: "* ]]; then
        numbers=$(echo "$line" | sed -n -e 's/sched: //p')
        echo "$numbers" >> "$output_file5"
    fi
done < "$input_file"


# plot times over

# extract times
# output_file="times_over.txt"
# while IFS= read -r line; do
#     if [[ $line == *"over sla: "* ]]; then
#         numbers=$(echo "$line" | sed -n -e 's/^.*over sla: \([^T]*\).*/\1/p')
#         echo "$numbers" >> "$output_file"
#     fi
# done < "$input_file"


# machine, proc type, sla, time over sla, time over actual comp
