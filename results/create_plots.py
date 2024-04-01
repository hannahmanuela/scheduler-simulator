import numpy as np
import pandas as pd
from matplotlib import pyplot as plt
import argparse

parser = argparse.ArgumentParser()

parser.add_argument('-variable-load', action='store_true')

args = parser.parse_args()

# load files
procs_added = pd.read_csv("procs_added.txt", index_col=None, names=["tick", "machineID", "procType", "sla", "actualComp", "migrated"])
procs_current = pd.read_csv("procs_current.txt", index_col=None, names=["tick", "machineID", "isActive", "sla", "actualComp", "compDone"])
procs_done = pd.read_csv("procs_done.txt", index_col=None, names=["tick", "machineID", "procType", "sla", "ticksPassed", "actualComp"])
procs_killed = pd.read_csv("procs_killed.txt", index_col=None, names=["tick", "machineID", "sla", "compDone", "memUsed"])

util_metrics = pd.read_csv("usage.txt", index_col=None, names=["tick", "machineID", "maxTicksPassedToSlaRatio", "memUsage", "qlen", "ticksInQ", "ticksLeftOver"])



# prepare
procs_current["compLeft"] = procs_current["sla"] - procs_current["compDone"]
procs_added["compLeft"] = procs_added["sla"]
all_procs = pd.concat([procs_current[['tick', "compLeft"]], procs_added[["tick", "compLeft"]]])
load_num_procs_per_tick = procs_added[["tick"]].groupby("tick").size().reset_index(name='numProcsCurrent')
load_work_per_tick = procs_added.groupby("tick").sum().reset_index()

procs_done['timePassedAsPct'] = (100 * procs_done["ticksPassed"]) / procs_done["sla"]

procs_late = procs_done.where(procs_done["ticksPassed"] > procs_done["sla"]).dropna().reset_index(drop=True)
procs_late = procs_late.where(procs_late["ticksPassed"] > procs_late["actualComp"]).dropna().reset_index(drop=True)

proc_timings = pd.merge(procs_done, load_num_procs_per_tick, on='tick', how='left')

ticks_left = util_metrics.groupby("tick")["ticksLeftOver"].agg(['min', 'max', 'mean']).reset_index()


# ==============================================================================================================
# Proc latency over load
# ==============================================================================================================

unique_ids = proc_timings['procType'].unique()

# Set up subplots
num_plots = len(unique_ids)
num_cols = 2  # Adjust as needed
num_rows = -(-num_plots // num_cols)  # Ceiling division

if args.variable_load:

    def get_percentile(pctile, proc_type):
        return proc_timings.where(proc_timings["procType"] == proc_type)[["numProcsCurrent", "timePassedAsPct"]].groupby("numProcsCurrent").quantile(pctile / 100).reset_index()

    fig, axes = plt.subplots(num_rows, num_cols, figsize=(15, 4*num_rows))
    if num_rows > 1 and num_cols > 1:
        axes = axes.flatten()

    # Plot each machine's data
    for i, id_ in enumerate(unique_ids):
        ax = axes[i]
        ax.scatter(get_percentile(50, id_)["numProcsCurrent"], get_percentile(50, id_)["timePassedAsPct"], color='lightblue', label="50")
        ax.plot(get_percentile(50, id_)["numProcsCurrent"], get_percentile(50, id_)["timePassedAsPct"], color='lightblue')

        ax.scatter(get_percentile(90, id_)["numProcsCurrent"], get_percentile(90, id_)["timePassedAsPct"], label="90")
        ax.plot(get_percentile(90, id_)["numProcsCurrent"], get_percentile(90, id_)["timePassedAsPct"])

        ax.scatter(get_percentile(95, id_)["numProcsCurrent"], get_percentile(95, id_)["timePassedAsPct"], color='grey', label="95")
        ax.plot(get_percentile(95, id_)["numProcsCurrent"], get_percentile(95, id_)["timePassedAsPct"], color='grey')

        ax.scatter(get_percentile(95, id_)["numProcsCurrent"], get_percentile(99, id_)["timePassedAsPct"], color='black', label="99")
        ax.plot(get_percentile(95, id_)["numProcsCurrent"], get_percentile(99, id_)["timePassedAsPct"], color='black')

        ax.set_title(f'Latency over load for proc type {id_}')
        ax.set_xlabel('Number of procs added that tick')
        ax.set_ylabel('Time passed as a fraction of the SLA')
        ax.legend()
        ax.grid(True)

    # If there are unused subplots, hide them
    for i in range(len(unique_ids), num_rows * num_cols):
        axes[i].axis('off')

    plt.tight_layout()

# ==============================================================================================================
# Proc latency distribution (hist)
# ==============================================================================================================

# Set up subplots
fig, axes = plt.subplots(num_rows, num_cols, figsize=(15, 4*num_rows))
if num_rows > 1 and num_cols > 1:
    axes = axes.flatten()

# Plot each machine's data
for i, id_ in enumerate(unique_ids):
    ax = axes[i]
    ax.hist(proc_timings.where(proc_timings["procType"] == id_)["timePassedAsPct"], bins=100)

    ax.set_title(f'Latency distribution for proc type {id_}')
    ax.set_xlabel('Time passed as a fraction of the SLA')
    ax.set_ylabel('Number of procs')
    ax.grid(True)

# If there are unused subplots, hide them
for i in range(len(unique_ids), num_rows * num_cols):
    axes[i].axis('off')

plt.tight_layout()


# 
# ticks added
# 

ticks_added = procs_added[["tick", "actualComp"]].groupby("tick").sum().reset_index()

plt.figure(figsize=(15,6))
plt.hist(ticks_added["actualComp"], bins=100)

plt.title('Distribution of ticks of compute added per tick')
plt.xlabel('Ticks added')
plt.ylabel('Frequency')
plt.grid(True)
plt.legend()

# ==============================================================================================================
# utilization
# ==============================================================================================================
plt.figure(figsize=(15,6))
plt.scatter(ticks_left["tick"], ticks_left["max"], label="max")
plt.plot(ticks_left["tick"], ticks_left["max"])

plt.scatter(ticks_left["tick"], ticks_left["min"], color='lightblue', label="min")
plt.plot(ticks_left["tick"], ticks_left["min"], color='lightblue')

plt.scatter(ticks_left["tick"], ticks_left["mean"], color='darkblue', label="mean")
plt.plot(ticks_left["tick"], ticks_left["mean"], color='darkblue')

plt.title('Ticks left over over time')
plt.xlabel('Tick')
plt.ylabel('Ticks left over')
plt.grid(True)
plt.legend()


# ==============================================================================================================
# q length range
# ==============================================================================================================
# q_lens = util_metrics.groupby("tick")["qlen"].agg(['min', 'max', 'mean']).reset_index()
# q_lens['range'] = q_lens['max'] - q_lens['min']

# plt.figure(figsize=(15,6))
# plt.scatter(q_lens["tick"], q_lens["range"], label="range")
# plt.plot(q_lens["tick"], q_lens["range"])

# # plt.scatter(q_lens["tick"], q_lens["min"], color='lightblue', label="min")
# # plt.plot(q_lens["tick"], q_lens["min"], color='lightblue')

# # plt.scatter(q_lens["tick"], q_lens["max"], color='darkblue', label="max")
# # plt.plot(q_lens["tick"], q_lens["max"], color='darkblue')

# plt.title('Qlength range')
# plt.xlabel('Tick')
# plt.ylabel('Range of qlen')
# plt.grid(True)
# plt.legend()




plt.show()

