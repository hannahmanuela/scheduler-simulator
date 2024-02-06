#!/bin/bash


input_file="results/out.txt"

# put proc done and proc created stats into different files
output_file1="results/procs_done.txt"
output_file2="results/procs_added.txt"
output_file3="results/procs_current.txt"
output_file4="results/procs_killed.txt"
output_file5="results/sched.txt"
output_file6="results/machines.txt"
output_file7="results/usage.txt"

# clear files first
>  "$output_file1"
>  "$output_file2"
>  "$output_file3"
>  "$output_file4"
>  "$output_file5"
>  "$output_file6"
>  "$output_file7"

# scrape output and separate data
while IFS= read -r line; do
    if [[ $line == *"done: "* ]]; then
        numbers=$(echo "$line" | sed -n -e 's/done: //p')
        echo "$numbers" >> "$output_file1"
    elif [[ $line == *"adding: "* ]]; then
        numbers=$(echo "$line" | sed -n -e 's/adding: //p')
        echo "$numbers" >> "$output_file2"
    elif [[ $line == *"current: "* ]]; then
        numbers=$(echo "$line" | sed -n -e 's/current: //p')
        echo "$numbers" >> "$output_file3"
    elif [[ $line == *"killing: "* ]]; then
        numbers=$(echo "$line" | sed -n -e 's/killing: //p')
        echo "$numbers" >> "$output_file4"
    elif [[ $line == *"sched: "* ]]; then
        numbers=$(echo "$line" | sed -n -e 's/sched: //p')
        echo "$numbers" >> "$output_file5"
    elif [[ $line == *"machine: "* ]]; then
        numbers=$(echo "$line" | sed -n -e 's/machine: //p')
        echo "$numbers" >> "$output_file6"
    elif [[ $line == *"usage: "* ]]; then
        numbers=$(echo "$line" | sed -n -e 's/usage: //p')
        echo "$numbers" >> "$output_file7"
    fi
done < "$input_file"

