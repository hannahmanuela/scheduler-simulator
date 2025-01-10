import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns  # Seaborn is a great library for creating scatter plots
import numpy as np

numMachines = 100
coresPerMachine = 8
totalMemoryPerMachine = 56000




# Load the data
ideal_usage_metrics = pd.read_csv("ideal_usage.txt", index_col=None, names=["nGenPerTick", "tick", "ticksLeftOver", "memFree"])
ideal_procs_done = pd.read_csv("ideal_procs_done.txt", index_col=None, names=["nGenPerTick", "price", "timePassed", "compDone"])

actual_usage_metrics = pd.read_csv("usage.txt", index_col=None, names=["nGenPerTick", "tick", "machineId", "begMemFree", "begMemPaged", "ticksLeftOver", "endMemFree", "endMemPaged", "sumMemSeen", "numProcsRan", "maxMemSeen", "fullQlen"])
actual_procs_done = pd.read_csv("procs_done.txt", index_col=None, names=["nGenPerTick", "machineId", "price", "timePassed", "compDone", "memory", "totMemPaged", "numTimesPaged"])

hermod_usage_metrics = pd.read_csv("hermod_usage.txt", index_col=None, names=["nGenPerTick", "tick", "machineId", "ticksLeftOver", "memFree"])
hermod_procs_done = pd.read_csv("hermod_procs_done.txt", index_col=None, names=["nGenPerTick", "price", "timePassed", "compDone"])

edf_usage_metrics = pd.read_csv("edf_usage.txt", index_col=None, names=["nGenPerTick", "tick", "ticksLeftOver", "memFree"])
edf_procs_done = pd.read_csv("edf_procs_done.txt", index_col=None, names=["nGenPerTick", "price", "timePassed", "compDone"])

# 1. Compute timeAsPercentage for both ideal and actual data
ideal_procs_done["timeAsPercentage"] = (ideal_procs_done["timePassed"] / ideal_procs_done["compDone"]) * 100
actual_procs_done["timeAsPercentage"] = (actual_procs_done["timePassed"] / actual_procs_done["compDone"]) * 100
hermod_procs_done["timeAsPercentage"] = (hermod_procs_done["timePassed"] / hermod_procs_done["compDone"]) * 100
edf_procs_done["timeAsPercentage"] = (edf_procs_done["timePassed"] / edf_procs_done["compDone"]) * 100

# # 2. Compute utilization for both ideal and actual data
ideal_usage_metrics["utilization"] = (numMachines * coresPerMachine - ideal_usage_metrics["ticksLeftOver"]) / (numMachines * coresPerMachine)
actual_usage_metrics["utilization"] = (coresPerMachine - actual_usage_metrics["ticksLeftOver"]) / (coresPerMachine)
hermod_usage_metrics["utilization"] = (coresPerMachine - hermod_usage_metrics["ticksLeftOver"]) / (coresPerMachine)
edf_usage_metrics["utilization"] = (numMachines * coresPerMachine - edf_usage_metrics["ticksLeftOver"]) / (numMachines * coresPerMachine)

# # 3. Compute memory utilization (1 - memFree / totalMemory)
# ideal_usage_metrics["mem_utilization"] = 1 - (ideal_usage_metrics["memFree"] / (numMachines * totalMemoryPerMachine))
# actual_usage_metrics["mem_utilization"] = 1 - (actual_usage_metrics["memFree"] / totalMemoryPerMachine)
# hermod_usage_metrics["mem_utilization"] = 1 - (hermod_usage_metrics["memFree"] / totalMemoryPerMachine)
# edf_usage_metrics["mem_utilization"] = 1 - (edf_usage_metrics["memFree"] / (numMachines * totalMemoryPerMachine))

print("num ideal finished: ", len(ideal_procs_done))
print("num actual finished: ", len(actual_procs_done))
print("num hermod finished: ", len(hermod_procs_done))
print("num edf finished: ", len(edf_procs_done))



ideal_usage_metrics = ideal_usage_metrics.where(ideal_usage_metrics['tick'] > 5).dropna()
ideal_usage_metrics['nGenPerTick'] = ideal_usage_metrics['nGenPerTick'].astype(int)
actual_usage_metrics = actual_usage_metrics.where(actual_usage_metrics['tick'] > 5).dropna()
actual_usage_metrics['nGenPerTick'] = actual_usage_metrics['nGenPerTick'].astype(int)
hermod_usage_metrics = hermod_usage_metrics.where(hermod_usage_metrics['tick'] > 5).dropna()
hermod_usage_metrics['nGenPerTick'] = hermod_usage_metrics['nGenPerTick'].astype(int)
edf_usage_metrics = edf_usage_metrics.where(edf_usage_metrics['tick'] > 5).dropna()
edf_usage_metrics['nGenPerTick'] = edf_usage_metrics['nGenPerTick'].astype(int)




# fig, ax = plt.subplots(3, 3, figsize=(9, 6))


# high_contrast_palette = ["#FF6347", "#1E90FF", "#32CD32", "#FFD700", "#00008B"]


# mine_percentiles = actual_procs_done.groupby(['nGenPerTick', 'price']).agg(
#     percentile_99=('timeAsPercentage', lambda x: np.percentile(x, 99))
# ).reset_index()

# sns.lineplot(data=mine_percentiles, x='nGenPerTick', y='percentile_99', hue='price', palette=high_contrast_palette, ax=ax[0,0])
# ax[0, 0].set_title("XX: Job latency as pct of runtime")
# ax[0, 0].set_ylabel("latency as pct of runtime")
# ax[0, 0].set_xlabel("load")
# ax[0, 0].grid(True)


# sns.boxplot(data=actual_usage_metrics, x='nGenPerTick', y='sumMemSeen', ax=ax[2, 0])
# ax[2, 0].set_title("XX: Distribution of sum mem seen")
# ax[2, 0].set_ylabel("total memory of all procs run that tick")
# ax[2, 0].set_xlabel("load")
# ax[2, 0].axhline(y=totalMemoryPerMachine, color='grey', linewidth=2, alpha=0.5)
# ax[2, 0].grid(True)




# mine_mem_paged = actual_procs_done.groupby(['nGenPerTick', 'price'])['numTimesPaged'].sum().reset_index()
# sns.lineplot(data=mine_mem_paged, x='nGenPerTick', y='numTimesPaged', hue='price', palette=high_contrast_palette, ax=ax[0, 2])
# ax[0, 2].set_title("XX: Num times paged")
# ax[0, 2].set_ylabel("times paged")
# ax[0, 2].set_xlabel("load")
# ax[0, 2].grid(True)



# mine_num_procs_pages = actual_procs_done.groupby(['nGenPerTick', 'price'])['numTimesPaged'].apply(lambda x: (x > 0).sum()).reset_index(name='countProcsPaged')
# sns.lineplot(data=mine_num_procs_pages, x='nGenPerTick', y='countProcsPaged', hue='price', palette=high_contrast_palette, ax=ax[1, 2])
# ax[1, 2].set_title("XX: Num procs paged")
# ax[1, 2].set_ylabel("procs paged")
# ax[1, 2].set_xlabel("load")
# ax[1, 2].grid(True)


# # ax[2, 2] -> distribution of mem paged per machine
# mine_mem_paged_per_machine = actual_procs_done.groupby(['nGenPerTick', 'machineId'])['totMemPaged'].sum().reset_index()
# sns.boxplot(data=mine_mem_paged_per_machine, x="nGenPerTick", y="totMemPaged", ax=ax[2, 2])
# ax[2, 2].set_title("mem paged per machine")
# ax[2, 2].set_xlabel("Load")
# ax[2, 2].set_ylabel("mem paged")
# ax[2, 2].grid(True)


# # num procs ran? run procs in q?
# actual_usage_metrics['numProcsDidntRun'] = actual_usage_metrics['fullQlen'] - actual_usage_metrics['numProcsRan']
# sns.boxplot(data=actual_usage_metrics, x='nGenPerTick', y='numProcsRan', ax=ax[2, 1])
# ax[2, 1].set_title("XX: Distribution of num procs ran")
# ax[2, 1].set_ylabel("num procs ran")
# ax[2, 1].set_xlabel("load")
# ax[2, 1].grid(True)


# sns.boxplot(data=actual_usage_metrics, x="nGenPerTick", y="utilization", ax=ax[1, 0])
# ax[1, 0].set_title("distribution of comp utilization")
# ax[1, 0].set_xlabel("Load")
# ax[1, 0].set_ylabel("comp utilization")
# ax[1, 0].grid(True)


# mine_mem_paged = actual_procs_done.groupby(['nGenPerTick', 'price'])['totMemPaged'].sum().reset_index()
# sns.lineplot(data=mine_mem_paged, x='nGenPerTick', y='totMemPaged', hue='price', palette=high_contrast_palette, ax=ax[0, 1])
# ax[0, 1].set_title("XX: Total memory paged")
# ax[0, 1].set_ylabel("memory paged")
# ax[0, 1].set_xlabel("load")
# ax[0, 1].grid(True)


# actual_usage_metrics['avgMemUsed'] = 1 - (((actual_usage_metrics['begMemFree'] + actual_usage_metrics['endMemFree']) / 2) / totalMemoryPerMachine)
# sns.boxplot(data=actual_usage_metrics, x="nGenPerTick", y="avgMemUsed", ax=ax[1, 1])
# ax[1, 1].set_title("distribution of mem utilization")
# ax[1, 1].set_xlabel("Load")
# ax[1, 1].set_ylabel("avg mem used")
# ax[1, 1].grid(True)
# plt.plot()
# plt.show()





# plots I will need to draw:

# =================================================================================
# hermod vs xx vs edf, latency
# =================================================================================

# fig, ax = plt.subplots(2, 4, figsize=(9, 6))

# high_contrast_palette = ["#FF6347", "#1E90FF", "#32CD32", "#FFD700", "#00008B"]

# hermod_percentiles = hermod_procs_done.groupby(['nGenPerTick', 'price']).agg(
#     percentile_99=('timeAsPercentage', lambda x: np.percentile(x, 99))
# ).reset_index()

# mine_percentiles = actual_procs_done.groupby(['nGenPerTick', 'price']).agg(
#     percentile_99=('timeAsPercentage', lambda x: np.percentile(x, 99))
# ).reset_index()

# ideal_percentiles = ideal_procs_done.groupby(['nGenPerTick', 'price']).agg(
#     percentile_99=('timeAsPercentage', lambda x: np.percentile(x, 99))
# ).reset_index()

# edf_percentiles = edf_procs_done.groupby(['nGenPerTick', 'price']).agg(
#     percentile_99=('timeAsPercentage', lambda x: np.percentile(x, 99))
# ).reset_index()


# sns.lineplot(data=hermod_percentiles, x='nGenPerTick', y='percentile_99', hue='price', palette=high_contrast_palette, ax=ax[0, 0])
# ax[0, 0].set_title("Hermod")
# ax[0, 0].set_xlabel("Load")
# ax[0, 0].set_ylabel("99 pctile latency as pct of runtime")
# ax[0, 0].grid(True)

# sns.boxplot(data=hermod_usage_metrics, x="nGenPerTick", y="utilization", ax=ax[1, 0])
# ax[1, 0].set_title("Hermod")
# ax[1, 0].set_xlabel("Load")
# ax[1, 0].set_ylabel("Distribution of compute util")
# ax[1, 0].grid(True)

# sns.lineplot(data=mine_percentiles, x='nGenPerTick', y='percentile_99', hue='price', palette=high_contrast_palette, ax=ax[0, 1])
# ax[0, 1].set_title("XX")
# ax[0, 1].set_ylabel("99 pctile latency as pct of runtime")
# ax[0, 1].set_xlabel("Load")
# ax[0, 1].grid(True)

# sns.boxplot(data=actual_usage_metrics, x="nGenPerTick", y="utilization", ax=ax[1, 1])
# ax[1, 1].set_title("XX")
# ax[1, 1].set_xlabel("Load")
# ax[1, 1].set_ylabel("Distribution of compute util")
# ax[1, 1].grid(True)

# sns.lineplot(data=ideal_percentiles, x='nGenPerTick', y='percentile_99', hue='price', palette=high_contrast_palette, ax=ax[0, 2])
# ax[0, 2].set_title("Ideal XX")
# ax[0, 2].set_ylabel("99 pctile latency as pct of runtime")
# ax[0, 2].set_xlabel("Load")
# ax[0, 2].grid(True)

# sns.boxplot(data=ideal_usage_metrics, x="nGenPerTick", y="utilization", ax=ax[1, 2])
# ax[1, 2].set_title("Ideal XX")
# ax[1, 2].set_xlabel("Load")
# ax[1, 2].set_ylabel("Distribution of compute util")
# ax[1, 2].grid(True)


# sns.lineplot(data=edf_percentiles, x='nGenPerTick', y='percentile_99', hue='price', palette=high_contrast_palette, ax=ax[0, 3])
# ax[0, 3].set_title("EDF")
# ax[0, 3].set_ylabel("99 pctile latency as pct of runtime")
# ax[0, 3].set_xlabel("Load")
# ax[0, 3].grid(True)

# sns.boxplot(data=edf_usage_metrics, x="nGenPerTick", y="utilization", ax=ax[1, 3])
# ax[1, 3].set_title("EDF")
# ax[1, 3].set_xlabel("Load")
# ax[1, 3].set_ylabel("Distribution of compute util")
# ax[1, 3].grid(True)

# y_max_0 = mine_percentiles["percentile_99"].max() * 1.05
# y_min_0 = -0.08 * y_max_0


# ax[0, 0].set_ylim(y_min_0, y_max_0)
# ax[0, 1].set_ylim(y_min_0, y_max_0)
# ax[0, 2].set_ylim(y_min_0, y_max_0)
# ax[0, 3].set_ylim(y_min_0, y_max_0)

# ax[1, 0].set_ylim(-0.1, 1.1)
# ax[1, 1].set_ylim(-0.1, 1.1)
# ax[1, 2].set_ylim(-0.1, 1.1)
# ax[1, 3].set_ylim(-0.1, 1.1)


# plt.tight_layout()
# plt.savefig('hermod_xx_edf_latency.png')
# plt.show()









# =================================================================================
# self comparison, mem paging/util
# =================================================================================



fig, ax = plt.subplots(3, 1, figsize=(6, 9))

high_contrast_palette = ["#FF6347", "#1E90FF", "#32CD32", "#FFD700", "#00008B"]

mine_num_procs_pages = actual_procs_done.groupby(['nGenPerTick', 'price'])['numTimesPaged'].apply(lambda x: (x > 0).sum()).reset_index(name='countProcsPaged')
sns.lineplot(data=mine_num_procs_pages, x='nGenPerTick', y='countProcsPaged', hue='price', palette=high_contrast_palette, ax=ax[0])
ax[0].set_title("Num procs swapped")
ax[0].set_ylabel("procs swapped")
ax[0].set_xlabel("load")
ax[0].grid(True)


# ax[2, 2] -> distribution of mem paged per machine
mine_mem_paged_per_machine = actual_procs_done.groupby(['nGenPerTick', 'machineId'])['totMemPaged'].sum().reset_index()
sns.boxplot(data=mine_mem_paged_per_machine, x="nGenPerTick", y="totMemPaged", ax=ax[1])
ax[1].set_title("Total Mem swapped per machine")
ax[1].set_xlabel("load")
ax[1].set_ylabel("mem swapped")
ax[1].grid(True)

actual_usage_metrics['avgMemUsed'] = 1 - (((actual_usage_metrics['begMemFree'] + actual_usage_metrics['endMemFree']) / 2) / totalMemoryPerMachine)
sns.boxplot(data=actual_usage_metrics, x="nGenPerTick", y="avgMemUsed", ax=ax[2])
ax[2].set_title("Distribution of mem utilization")
ax[2].set_xlabel("Load")
ax[2].set_ylabel("mem utilization")
ax[2].grid(True)

plt.tight_layout()
plt.savefig('memory_graphs.png')
plt.show()


