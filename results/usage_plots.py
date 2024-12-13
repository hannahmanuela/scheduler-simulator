import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns  # Seaborn is a great library for creating scatter plots

numMachines = 10
coresPerMachine = 8

# Load the data
ideal_usage_metrics = pd.read_csv("ideal_usage.txt", index_col=None, names=["nGenPerTick", "tick", "ticksLeftOver"])
ideal_procs_done = pd.read_csv("ideal_procs_done.txt", index_col=None, names=["nGenPerTick", "willingToSpend", "timePassed", "compDone"])

# 1. Compute timeAsPercentage as (timePassed / compDone) * 100
ideal_procs_done["timeAsPercentage"] = (ideal_procs_done["timePassed"] / ideal_procs_done["compDone"]) * 100

ideal_usage_metrics["utilization"] = (numMachines * coresPerMachine - ideal_usage_metrics["ticksLeftOver"]) / (numMachines * coresPerMachine)

# Create the figure with two subplots, stacked vertically
fig, ax = plt.subplots(2, 1, figsize=(10, 10), sharex=True)

# 2. First subplot: Scatter plot for timeAsPercentage
high_contrast_palette = ["#FF6347", "#1E90FF", "#32CD32", "#FFD700"]  # Example: Red, Blue, Green, Yellow

# Create the stripplot with the custom high-contrast palette
sns.stripplot(data=ideal_procs_done, x="nGenPerTick", y="timeAsPercentage", hue="willingToSpend", 
              jitter=True, palette=high_contrast_palette, ax=ax[0])

# Labeling the first subplot
ax[0].set_title("Time Passed as % of Completion (compDone) by Willing to Spend")
ax[0].set_ylabel("Time Passed as % of compDone")

# 3. Second subplot: Violin plot for the distribution of ticksLeftOver (usage)
sns.violinplot(data=ideal_usage_metrics, x="nGenPerTick", y="utilization", ax=ax[1])

# Labeling the second subplot
ax[1].set_title("Distribution of Utilization by nGenPerTick")
ax[1].set_xlabel("nGenPerTick")
ax[1].set_ylabel("Utilization")

# Display the plot with aligned x-axes
plt.tight_layout()
plt.savefig('current_res.png')
plt.show()
