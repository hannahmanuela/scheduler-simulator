import pandas as pd
from matplotlib import pyplot as plt
import seaborn as sns


util_metrics = pd.read_csv("usage.txt", index_col=None, names=["nGenPerTick", "tick", "machineID", "qlen", "ticksLeftOver"])
ideal_metrics = pd.read_csv("ideal_usage.txt", index_col=None, names=["nGenPerTick", "tick", "machineID", "qlen", "ticksLeftOver"])
said_no = pd.read_csv("said_no.txt", index_col=None, names=["nGenPerTick", "tick", "deadline"])
ideal_said_no = pd.read_csv("ideal_said_no.txt", index_col=None, names=["nGenPerTick", "tick", "deadline"])
procs_created = pd.read_csv("procs_created.txt", index_col=None, names=["nGenPerTick", "tick", "deadline"])

util_metrics = util_metrics.where(util_metrics["tick"] > 100).dropna()
ideal_metrics = ideal_metrics.where(ideal_metrics["tick"] > 100).dropna()
said_no = said_no.where(said_no["tick"] > 100).dropna()
ideal_said_no = ideal_said_no.where(said_no["tick"] > 100).dropna()
procs_created = procs_created.where(procs_created["tick"] > 100).dropna()

util_metrics_grouped = util_metrics.groupby('nGenPerTick')['ticksLeftOver'].agg(['min', 'max', 'mean']).reset_index()
ideal_metrics_grouped = ideal_metrics.groupby('nGenPerTick')['ticksLeftOver'].agg(['min', 'max', 'mean']).reset_index()

procs_created_grouped = procs_created.groupby(["nGenPerTick", "deadline"]).size().reset_index(name="count_created")

said_no_grouped = said_no.groupby(['nGenPerTick', 'deadline']).size().reset_index(name='count_rejected')
relative_said_no = pd.merge(procs_created_grouped, said_no_grouped, on=['nGenPerTick', 'deadline'], how='outer').fillna(0)
relative_said_no['pct_rejected'] = relative_said_no['count_rejected'] / relative_said_no['count_created']

ideal_said_no_grouped = ideal_said_no.groupby(['nGenPerTick', 'deadline']).size().reset_index(name='count_rejected')
ideal_relative_said_no = pd.merge(procs_created_grouped, ideal_said_no_grouped, on=['nGenPerTick', 'deadline'], how='outer').fillna(0)
ideal_relative_said_no['pct_rejected'] = ideal_relative_said_no['count_rejected'] / ideal_relative_said_no['count_created']


# Create 4 subplots
fig, axs = plt.subplots(2, 2, sharex=True, figsize=(14,8))

# Subplot 1: min, max, mean of ticksLeftOver (real data)
sns.lineplot(data=util_metrics_grouped, x='nGenPerTick', y='min', label='Min', ax=axs[0, 0], color='red')
sns.scatterplot(data=util_metrics_grouped, x='nGenPerTick', y='min', color='red', s=100, ax=axs[0, 0])

sns.lineplot(data=util_metrics_grouped, x='nGenPerTick', y='max', label='Max', ax=axs[0, 0], color='blue')
sns.scatterplot(data=util_metrics_grouped, x='nGenPerTick', y='max', color='blue', s=100, ax=axs[0, 0])

sns.lineplot(data=util_metrics_grouped, x='nGenPerTick', y='mean', label='Mean', ax=axs[0, 0], color='green')
sns.scatterplot(data=util_metrics_grouped, x='nGenPerTick', y='mean', color='green', s=100, ax=axs[0, 0])

axs[0, 0].set_title('Ticks LeftOver (Min, Max, Mean) vs nGenPerTick (Real Data)')
axs[0, 0].set_xlabel('nGenPerTick')
axs[0, 0].set_ylabel('ticksLeftOver')
axs[0, 0].set_xticks(util_metrics_grouped['nGenPerTick'])
axs[0, 0].legend()

# Subplot 2: Percent of rejected procs (real data)
deadline_colors = {
    1: 'red',     # Replace with your own deadline names and colors
    4: 'orange',
    100: 'green',
    1000: 'blue',
}
sns.scatterplot(data=relative_said_no, x='nGenPerTick', y='pct_rejected', hue='deadline', palette=deadline_colors, ax=axs[1, 0])
sns.lineplot(data=relative_said_no, x='nGenPerTick', y='pct_rejected', ax=axs[1, 0], hue='deadline', palette=deadline_colors, legend=False)

axs[1, 0].set_title('Percent of created procs rejected (Real Data)')
axs[1, 0].set_xlabel('nGenPerTick')
axs[1, 0].set_ylabel('Pct rejected')
axs[1, 0].legend()

# Subplot 3: min, max, mean of ticksLeftOver (ideal data)
sns.lineplot(data=ideal_metrics_grouped, x='nGenPerTick', y='min', label='Min', ax=axs[0, 1], color='red')
sns.scatterplot(data=ideal_metrics_grouped, x='nGenPerTick', y='min', color='red', s=100, ax=axs[0, 1])

sns.lineplot(data=ideal_metrics_grouped, x='nGenPerTick', y='max', label='Max', ax=axs[0, 1], color='blue')
sns.scatterplot(data=ideal_metrics_grouped, x='nGenPerTick', y='max', color='blue', s=100, ax=axs[0, 1])

sns.lineplot(data=ideal_metrics_grouped, x='nGenPerTick', y='mean', label='Mean', ax=axs[0, 1], color='green')
sns.scatterplot(data=ideal_metrics_grouped, x='nGenPerTick', y='mean', color='green', s=100, ax=axs[0, 1])

axs[0, 1].set_title('Ticks LeftOver (Min, Max, Mean) vs nGenPerTick (Ideal Data)')
axs[0, 1].set_xlabel('nGenPerTick')
axs[0, 1].set_ylabel('ticksLeftOver')
axs[0, 1].legend()

# Subplot 4: Percent of rejected procs (ideal data)
sns.scatterplot(data=ideal_relative_said_no, x='nGenPerTick', y='pct_rejected', hue='deadline', palette=deadline_colors, ax=axs[1, 1])
sns.lineplot(data=ideal_relative_said_no, x='nGenPerTick', y='pct_rejected', ax=axs[1, 1], hue='deadline', palette=deadline_colors, legend=False)

axs[1, 1].set_title('Percent of created procs rejected (Ideal Data)')
axs[1, 1].set_xlabel('nGenPerTick')
axs[1, 1].set_ylabel('Pct rejected')
axs[1, 1].legend()

# Adjust layout and show plot
plt.tight_layout()
plt.show()


