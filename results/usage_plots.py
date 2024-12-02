import pandas as pd
from matplotlib import pyplot as plt
import seaborn as sns

numMachines = 10
numCores = 8

# Read your data files
util_metrics = pd.read_csv("usage.txt", index_col=None, names=["nGenPerTick", "tick", "machineID", "qlen", "ticksLeftOver"])
ideal_metrics = pd.read_csv("ideal_usage.txt", index_col=None, names=["nGenPerTick", "tick", "machineID", "qlen", "ticksLeftOver"])
said_no = pd.read_csv("said_no.txt", index_col=None, names=["nGenPerTick", "tick", "deadline"])
ideal_said_no = pd.read_csv("ideal_said_no.txt", index_col=None, names=["nGenPerTick", "tick", "deadline"])
procs_created = pd.read_csv("procs_created.txt", index_col=None, names=["nGenPerTick", "tick", "deadline"])

# Create utilization metrics
util_metrics["utilization"] = (numCores - util_metrics["ticksLeftOver"]) / numCores
ideal_metrics["utilization"] = ((numMachines * numCores) - ideal_metrics["ticksLeftOver"]) / (numMachines * numCores)

# Group and merge the data
procs_created_grouped = procs_created.groupby(["nGenPerTick", "deadline"]).size().reset_index(name="count_created")
said_no_grouped = said_no.groupby(['nGenPerTick', 'deadline']).size().reset_index(name='count_rejected')
relative_said_no = pd.merge(procs_created_grouped, said_no_grouped, on=['nGenPerTick', 'deadline'], how='outer').fillna(0)
relative_said_no['pct_rejected'] = relative_said_no['count_rejected'] / relative_said_no['count_created']

ideal_said_no_grouped = ideal_said_no.groupby(['nGenPerTick', 'deadline']).size().reset_index(name='count_rejected')
ideal_relative_said_no = pd.merge(procs_created_grouped, ideal_said_no_grouped, on=['nGenPerTick', 'deadline'], how='outer').fillna(0)
ideal_relative_said_no['pct_rejected'] = ideal_relative_said_no['count_rejected'] / ideal_relative_said_no['count_created']

# Create 4 subplots with shared x-axis
fig, axs = plt.subplots(2, 2, figsize=(14, 8), sharey=True)

# Subplot 1: min, max, mean of ticksLeftOver (real data)
sns.violinplot(data=util_metrics, x='nGenPerTick', y='utilization', ax=axs[0, 0])
axs[0, 0].set_title('REAL - Dist of utilization for each nGenPerTick')
axs[0, 0].set_xlabel('nGenPerTick')
axs[0, 0].set_ylabel('Utilization')

# Subplot 2: Percent of rejected procs (real data)
deadline_colors = {
    1: 'red',     # Replace with your own deadline names and colors
    4: 'orange',
    100: 'green',
    1000: 'blue',
}
sns.scatterplot(data=relative_said_no, x='nGenPerTick', y='pct_rejected', hue='deadline', palette=deadline_colors, ax=axs[1, 0])
sns.lineplot(data=relative_said_no, x='nGenPerTick', y='pct_rejected', ax=axs[1, 0], hue='deadline', palette=deadline_colors, legend=False)

axs[1, 0].set_title('REAL - Percent of created procs rejected')
axs[1, 0].set_xlabel('nGenPerTick')
axs[1, 0].set_ylabel('Pct rejected')

# Subplot 3: min, max, mean of ticksLeftOver (ideal data)
sns.violinplot(data=ideal_metrics, x='nGenPerTick', y='utilization', ax=axs[0, 1])
axs[0, 1].set_title('IDEAL - Dist of utilization for each nGenPerTick')
axs[0, 1].set_xlabel('nGenPerTick')
axs[0, 1].set_ylabel('Utilization')

# Subplot 4: Percent of rejected procs (ideal data)
sns.scatterplot(data=ideal_relative_said_no, x='nGenPerTick', y='pct_rejected', hue='deadline', palette=deadline_colors, ax=axs[1, 1])
sns.lineplot(data=ideal_relative_said_no, x='nGenPerTick', y='pct_rejected', ax=axs[1, 1], hue='deadline', palette=deadline_colors, legend=False)

axs[1, 1].set_title('IDEAL - Percent of created procs rejected')
axs[1, 1].set_xlabel('nGenPerTick')
axs[1, 1].set_ylabel('Pct rejected')

# Rotate x-axis labels for all subplots
for ax in axs.flat:
    ax.tick_params(axis='x', rotation=45)

# Adjust layout and show plot
plt.tight_layout()
plt.show()

plt.savefig('current_res.png')
