import pandas as pd
from matplotlib import pyplot as plt
import seaborn as sns


util_metrics = pd.read_csv("usage.txt", index_col=None, names=["nGenPerTick", "tick", "machineID", "qlen", "ticksLeftOver"])
said_no = pd.read_csv("said_no.txt", index_col=None, names=["nGenPerTick", "deadline"])

util_metrics = util_metrics.where(util_metrics["tick"] > 100).dropna()

util_metrics_grouped = util_metrics.groupby('nGenPerTick')['ticksLeftOver'].agg(['min', 'max', 'mean']).reset_index()

said_no_grouped = said_no.groupby(['nGenPerTick', 'deadline']).size().reset_index(name='count')
# print(said_no_grouped)


fig, axs = plt.subplots(2, 1, figsize=(8, 6))

# Subplot 1: min, max, mean of ticksLeftOver
sns.lineplot(data=util_metrics_grouped, x='nGenPerTick', y='min', label='Min', ax=axs[0], color='red')
sns.scatterplot(data=util_metrics_grouped, x='nGenPerTick', y='min', color='red', s=100, ax=axs[0])

sns.lineplot(data=util_metrics_grouped, x='nGenPerTick', y='max', label='Max', ax=axs[0], color='blue')
sns.scatterplot(data=util_metrics_grouped, x='nGenPerTick', y='max', color='blue', s=100, ax=axs[0])

sns.lineplot(data=util_metrics_grouped, x='nGenPerTick', y='mean', label='Mean', ax=axs[0], color='green')
sns.scatterplot(data=util_metrics_grouped, x='nGenPerTick', y='mean', color='green', s=100, ax=axs[0])


axs[0].set_title('Ticks LeftOver (Min, Max, Mean) vs nGenPerTick')
axs[0].set_xlabel('nGenPerTick')
axs[0].set_ylabel('ticksLeftOver')
axs[0].legend()

# Subplot 2: Number of rows in said_no colored by deadline
deadline_colors = {
    1: 'red',     # Replace with your own deadline names and colors
    4: 'orange',
    100: 'green',
}
sns.scatterplot(data=said_no_grouped, x='nGenPerTick', y='count', hue='deadline', palette=deadline_colors, ax=axs[1])
sns.lineplot(data=said_no_grouped, x='nGenPerTick', y='count', ax=axs[1], hue='deadline', palette=deadline_colors, legend=False)

axs[1].set_title('Rows in said_no vs nGenPerTick')
axs[1].set_xlabel('nGenPerTick')
axs[1].set_ylabel('Number of Rows')
axs[1].legend()

# Show plot
plt.tight_layout()
plt.show()



