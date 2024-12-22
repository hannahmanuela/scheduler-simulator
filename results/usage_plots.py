import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns  # Seaborn is a great library for creating scatter plots

numMachines = 10
coresPerMachine = 8

# Load the data
ideal_usage_metrics = pd.read_csv("ideal_usage.txt", index_col=None, names=["nGenPerTick", "tick", "ticksLeftOver"])
ideal_procs_done = pd.read_csv("ideal_procs_done.txt", index_col=None, names=["nGenPerTick", "willingToSpend", "timePassed", "compDone"])

actual_usage_metrics = pd.read_csv("usage.txt", index_col=None, names=["nGenPerTick", "tick", "machineId", "ticksLeftOver", "memFree"])
actual_procs_done = pd.read_csv("procs_done.txt", index_col=None, names=["nGenPerTick", "willingToSpend", "timePassed", "compDone", "timeQedAtGS"])

# 1. Compute timeAsPercentage for both ideal and actual data
ideal_procs_done["timeAsPercentage"] = (ideal_procs_done["timePassed"] / ideal_procs_done["compDone"]) * 100
actual_procs_done["timeAsPercentage"] = (actual_procs_done["timePassed"] / actual_procs_done["compDone"]) * 100

# 2. Compute utilization for both ideal and actual data
ideal_usage_metrics["utilization"] = (numMachines * coresPerMachine - ideal_usage_metrics["ticksLeftOver"]) / (numMachines * coresPerMachine)
actual_usage_metrics["utilization"] = (coresPerMachine - actual_usage_metrics["ticksLeftOver"]) / (coresPerMachine)

# Create the figure with four subplots (2 rows, 2 columns)
fig, ax = plt.subplots(2, 2, figsize=(14, 10), sharex=True)

# High contrast color palette
high_contrast_palette = ["#FF6347", "#1E90FF", "#32CD32", "#FFD700", "#00008B"]

# 3. First subplot: Strip plot for Time Passed as % of Completion for Ideal data
sns.stripplot(data=ideal_procs_done, x="nGenPerTick", y="timeAsPercentage", hue="willingToSpend", 
              jitter=True, palette=high_contrast_palette, ax=ax[0, 0])

ax[0, 0].set_title("Ideal: Time Passed as % of Completion by Willing to Spend")
ax[0, 0].set_ylabel("Time Passed as % of compDone")

# 4. Second subplot: Violin plot for Utilization for Ideal data
sns.boxplot(data=ideal_usage_metrics, x="nGenPerTick", y="utilization", ax=ax[1, 0])

ax[1, 0].set_title("Ideal: Distribution of Utilization by nGenPerTick")
ax[1, 0].set_xlabel("nGenPerTick")
ax[1, 0].set_ylabel("Utilization")

# 5. Third subplot: Strip plot for Time Passed as % of Completion for Actual data
sns.stripplot(data=actual_procs_done, x="nGenPerTick", y="timeAsPercentage", hue="willingToSpend", 
              jitter=True, palette=high_contrast_palette, ax=ax[0, 1])

ax[0, 1].set_title("Actual: Time Passed as % of Completion by Willing to Spend")
ax[0, 1].set_ylabel("Time Passed as % of compDone")

# 6. Fourth subplot: Violin plot for Utilization for Actual data
sns.boxplot(data=actual_usage_metrics, x="nGenPerTick", y="utilization", ax=ax[1, 1])

ax[1, 1].set_title("Actual: Distribution of Utilization by nGenPerTick")
ax[1, 1].set_xlabel("nGenPerTick")
ax[1, 1].set_ylabel("Utilization")

# Adjust the layout to avoid overlap
plt.tight_layout()

# Save the figure and show it
plt.savefig('combined_res.png')
plt.show()
