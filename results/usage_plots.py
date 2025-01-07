import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns  # Seaborn is a great library for creating scatter plots
import numpy as np

numMachines = 100
coresPerMachine = 8
totalMemoryPerMachine = 64000




# Load the data
ideal_usage_metrics = pd.read_csv("ideal_usage.txt", index_col=None, names=["nGenPerTick", "tick", "ticksLeftOver", "memFree"])
ideal_procs_done = pd.read_csv("ideal_procs_done.txt", index_col=None, names=["nGenPerTick", "price", "timePassed", "compDone"])

actual_usage_metrics = pd.read_csv("usage.txt", index_col=None, names=["nGenPerTick", "tick", "machineId", "ticksLeftOver", "memFree"])
actual_procs_done = pd.read_csv("procs_done.txt", index_col=None, names=["nGenPerTick", "price", "timePassed", "compDone"])

hermod_usage_metrics = pd.read_csv("hermod_usage.txt", index_col=None, names=["nGenPerTick", "tick", "machineId", "ticksLeftOver", "memFree"])
hermod_procs_done = pd.read_csv("hermod_procs_done.txt", index_col=None, names=["nGenPerTick", "price", "timePassed", "compDone"])

edf_usage_metrics = pd.read_csv("edf_usage.txt", index_col=None, names=["nGenPerTick", "tick", "machineId", "ticksLeftOver", "memFree"])
edf_procs_done = pd.read_csv("edf_procs_done.txt", index_col=None, names=["nGenPerTick", "price", "timePassed", "compDone"])

# 1. Compute timeAsPercentage for both ideal and actual data
ideal_procs_done["timeAsPercentage"] = (ideal_procs_done["timePassed"] / ideal_procs_done["compDone"]) * 100
actual_procs_done["timeAsPercentage"] = (actual_procs_done["timePassed"] / actual_procs_done["compDone"]) * 100
hermod_procs_done["timeAsPercentage"] = (hermod_procs_done["timePassed"] / hermod_procs_done["compDone"]) * 100
edf_procs_done["timeAsPercentage"] = (edf_procs_done["timePassed"] / edf_procs_done["compDone"]) * 100

# 2. Compute utilization for both ideal and actual data
ideal_usage_metrics["utilization"] = (numMachines * coresPerMachine - ideal_usage_metrics["ticksLeftOver"]) / (numMachines * coresPerMachine)
actual_usage_metrics["utilization"] = (coresPerMachine - actual_usage_metrics["ticksLeftOver"]) / (coresPerMachine)
hermod_usage_metrics["utilization"] = (coresPerMachine - hermod_usage_metrics["ticksLeftOver"]) / (coresPerMachine)
edf_usage_metrics["utilization"] = (numMachines * coresPerMachine - edf_usage_metrics["ticksLeftOver"]) / (numMachines * coresPerMachine)

# 3. Compute memory utilization (1 - memFree / totalMemory)
ideal_usage_metrics["mem_utilization"] = 1 - (ideal_usage_metrics["memFree"] / (numMachines * totalMemoryPerMachine))
actual_usage_metrics["mem_utilization"] = 1 - (actual_usage_metrics["memFree"] / totalMemoryPerMachine)
hermod_usage_metrics["mem_utilization"] = 1 - (hermod_usage_metrics["memFree"] / totalMemoryPerMachine)
edf_usage_metrics["mem_utilization"] = 1 - (edf_usage_metrics["memFree"] / (numMachines * totalMemoryPerMachine))

print("num ideal finished: ", len(ideal_procs_done))
print("num actual finished: ", len(actual_procs_done))
print("num hermod finished: ", len(hermod_procs_done))
print("num edf finished: ", len(edf_procs_done))






# plots I will need to draw:

# ===========================
# ideal vs edf, latency
# ===========================

fig, ax = plt.subplots(2, 2, figsize=(9, 6), sharex=True)

high_contrast_palette = ["#FF6347", "#1E90FF", "#32CD32", "#FFD700", "#00008B"]


ideal_percentiles = ideal_procs_done.groupby(['nGenPerTick', 'price']).agg(
    percentile_99=('timeAsPercentage', lambda x: np.percentile(x, 99))
).reset_index()

edf_percentiles = edf_procs_done.groupby(['nGenPerTick', 'price']).agg(
    percentile_99=('timeAsPercentage', lambda x: np.percentile(x, 99))
).reset_index()


sns.lineplot(data=ideal_percentiles, x='nGenPerTick', y='percentile_99', hue='price', palette=high_contrast_palette, ax=ax[0, 0])
ax[0, 0].set_title("Priority: 99 pctile job latency as pct of runtime")
ax[0, 0].set_ylabel("latency as pct of runtime")


sns.lineplot(data=edf_percentiles, x='nGenPerTick', y='percentile_99', hue='price', palette=high_contrast_palette, ax=ax[0, 1])
ax[0, 1].set_title("EDF: 99 pctile job latency as pct of runtime")
ax[0, 1].set_ylabel("latency as pct of runtime")


ideal_percentile_runtime = ideal_procs_done.groupby(['nGenPerTick', 'price']).agg(
    percentile_99=('compDone', lambda x: np.percentile(x, 99))
).reset_index()

edf_percentile_runtime = edf_procs_done.groupby(['nGenPerTick', 'price']).agg(
    percentile_99=('compDone', lambda x: np.percentile(x, 99))
).reset_index()

# print(edf_num_done)

sns.lineplot(data=ideal_percentile_runtime, x='nGenPerTick', y='percentile_99', hue='price', palette=high_contrast_palette, ax=ax[1, 0])
ax[1, 0].set_title("Priority: 99th pctile runtime")
ax[1, 0].set_ylabel("runtime")
ax[1, 0].set_xlabel("load")


sns.lineplot(data=edf_percentile_runtime, x='nGenPerTick', y='percentile_99', hue='price', palette=high_contrast_palette, ax=ax[1, 1])
ax[1, 1].set_title("EDF: 99th pctile runtime")
ax[1, 1].set_ylabel("runtime")
ax[1, 1].set_xlabel("load")


plt.tight_layout()
plt.savefig('ideal_edf_latency.png')
plt.show()



# ===========================
# hermod vs mine, latency
# ===========================


# fig, ax = plt.subplots(2, 1, figsize=(9, 6), sharex=True)

# high_contrast_palette = ["#FF6347", "#1E90FF", "#32CD32", "#FFD700", "#00008B"]


# hermod_percentiles = hermod_procs_done.groupby(['nGenPerTick', 'price']).agg(
#     percentile_99=('timeAsPercentage', lambda x: np.percentile(x, 99))
# ).reset_index()

# mine_percentiles = actual_procs_done.groupby(['nGenPerTick', 'price']).agg(
#     percentile_99=('timeAsPercentage', lambda x: np.percentile(x, 99))
# ).reset_index()

# sns.lineplot(data=hermod_percentiles, x='nGenPerTick', y='percentile_99', hue='price', palette=high_contrast_palette, ax=ax[0])
# ax[0].set_title("Priority: Job latency as pct of runtime")
# ax[0].set_ylabel("latency as pct of runtime")


# sns.lineplot(data=mine_percentiles, x='nGenPerTick', y='percentile_99', hue='price', palette=high_contrast_palette, ax=ax[1])
# ax[1].set_title("EDF: Job latency as pct of runtime")
# ax[1].set_ylabel("latency as pct of runtime")


# plt.tight_layout()
# plt.savefig('hermod_xx_latency.png')
# plt.show()




# ===========================
# ideal vs mine, latency & util
# ===========================

# fig, ax = plt.subplots(3, 2, figsize=(12, 9), sharex=True)

# high_contrast_palette = ["#FF6347", "#1E90FF", "#32CD32", "#FFD700", "#00008B"]


# sns.stripplot(data=ideal_procs_done, x="nGenPerTick", y="timeAsPercentage", hue="price", 
#               jitter=True, palette=high_contrast_palette, ax=ax[0, 0])
# ax[0, 0].set_title("Ideal: Job latency as pct of runtime")
# ax[0, 0].set_ylabel("Time Passed as % of compDone")


# sns.boxplot(data=ideal_usage_metrics, x="nGenPerTick", y="utilization", ax=ax[1, 0])
# ax[1, 0].set_title("Ideal: Distribution of Compute Utilization")
# ax[1, 0].set_xlabel("Load")
# ax[1, 0].set_ylabel("Compute util")
# ax[1, 0].axhline(y=1, color='grey', linewidth=2, alpha=0.5)


# sns.stripplot(data=actual_procs_done, x="nGenPerTick", y="timeAsPercentage", hue="price", 
#               jitter=True, palette=high_contrast_palette, ax=ax[0, 1])
# ax[0, 1].set_title("Actual: Job latency as pct of runtime")
# ax[0, 1].set_ylabel("Time Passed as % of compDone")


# sns.boxplot(data=actual_usage_metrics, x="nGenPerTick", y="utilization", ax=ax[1, 1])
# ax[1, 1].set_title("Actual: Distribution of Compute Utilization")
# ax[1, 1].set_xlabel("Load")
# ax[1, 1].set_ylabel("Compute util")
# ax[1, 1].axhline(y=1, color='grey', linewidth=2, alpha=0.5)


# sns.boxplot(data=ideal_usage_metrics, x="nGenPerTick", y="mem_utilization", ax=ax[2, 0])
# ax[2, 0].set_title("Ideal: Distribution of Memory Utilization")
# ax[2, 0].set_xlabel("Load")
# ax[2, 0].set_ylabel("Memory util")
# ax[2, 0].axhline(y=1, color='grey', linewidth=2, alpha=0.5)


# sns.boxplot(data=actual_usage_metrics, x="nGenPerTick", y="mem_utilization", ax=ax[2, 1])
# ax[2, 1].set_title("Actual: Distribution of Memory Utilization")
# ax[2, 1].set_xlabel("Load")
# ax[2, 1].set_ylabel("Memory util")
# ax[2, 1].axhline(y=1, color='grey', linewidth=2, alpha=0.5)



# y_max_0 = max(ideal_procs_done["timeAsPercentage"].max(), actual_procs_done["timeAsPercentage"].max()) * 1.1
# y_min_0 = -0.08 * y_max_0


# ax[0, 0].set_ylim(y_min_0, y_max_0)
# ax[0, 1].set_ylim(y_min_0, y_max_0)

# ax[1, 0].set_ylim(-0.1, 1.1)
# ax[1, 1].set_ylim(-0.1, 1.1)

# ax[2, 0].set_ylim(-0.1, 1.1)
# ax[2, 1].set_ylim(-0.1, 1.1)

# plt.tight_layout()
# plt.savefig('ideal_vs_mine.png')
# plt.show()
