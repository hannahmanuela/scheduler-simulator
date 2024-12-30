import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns  # Seaborn is a great library for creating scatter plots

numMachines = 100
coresPerMachine = 8
totalMemoryPerMachine = 64000

# Load the data
ideal_usage_metrics = pd.read_csv("ideal_usage.txt", index_col=None, names=["nGenPerTick", "tick", "ticksLeftOver", "memFree"])
ideal_procs_done = pd.read_csv("ideal_procs_done.txt", index_col=None, names=["nGenPerTick", "willingToSpend", "timePassed", "compDone"])

actual_usage_metrics = pd.read_csv("usage.txt", index_col=None, names=["nGenPerTick", "tick", "machineId", "ticksLeftOver", "memFree"])
actual_procs_done = pd.read_csv("procs_done.txt", index_col=None, names=["nGenPerTick", "willingToSpend", "timePassed", "compDone", "timeQedAtGS"])

# 1. Compute timeAsPercentage for both ideal and actual data
ideal_procs_done["timeAsPercentage"] = (ideal_procs_done["timePassed"] / ideal_procs_done["compDone"]) * 100
actual_procs_done["timeAsPercentage"] = (actual_procs_done["timePassed"] / actual_procs_done["compDone"]) * 100

# 2. Compute utilization for both ideal and actual data
ideal_usage_metrics["utilization"] = (numMachines * coresPerMachine - ideal_usage_metrics["ticksLeftOver"]) / (numMachines * coresPerMachine)
actual_usage_metrics["utilization"] = (coresPerMachine - actual_usage_metrics["ticksLeftOver"]) / (coresPerMachine)

# 3. Compute memory utilization (1 - memFree / totalMemory)
ideal_usage_metrics["mem_utilization"] = 1 - (ideal_usage_metrics["memFree"] / (numMachines * totalMemoryPerMachine))
actual_usage_metrics["mem_utilization"] = 1 - (actual_usage_metrics["memFree"] / totalMemoryPerMachine)

print("num ideal finished: ", len(ideal_procs_done))
print("num actual finished: ", len(actual_procs_done))


# Create the figure with six subplots (3 rows, 2 columns)
fig, ax = plt.subplots(3, 2, figsize=(12, 9), sharex=True)

# High contrast color palette
high_contrast_palette = ["#FF6347", "#1E90FF", "#32CD32", "#FFD700", "#00008B"]

# 4. First subplot: Strip plot for Time Passed as % of Completion for Ideal data
sns.stripplot(data=ideal_procs_done, x="nGenPerTick", y="timeAsPercentage", hue="willingToSpend", 
              jitter=True, palette=high_contrast_palette, ax=ax[0, 0])

ax[0, 0].set_title("Ideal: Time Passed as % of Completion by Willing to Spend")
ax[0, 0].set_ylabel("Time Passed as % of compDone")

# 5. Second subplot: Violin plot for Utilization for Ideal data
sns.boxplot(data=ideal_usage_metrics, x="nGenPerTick", y="utilization", ax=ax[1, 0])

ax[1, 0].set_title("Ideal: Distribution of Utilization by nGenPerTick")
ax[1, 0].set_xlabel("nGenPerTick")
ax[1, 0].set_ylabel("Utilization")
ax[1, 0].axhline(y=1, color='grey', linewidth=2, alpha=0.5)

# 6. Third subplot: Strip plot for Time Passed as % of Completion for Actual data
sns.stripplot(data=actual_procs_done, x="nGenPerTick", y="timeAsPercentage", hue="willingToSpend", 
              jitter=True, palette=high_contrast_palette, ax=ax[0, 1])

ax[0, 1].set_title("Actual: Time Passed as % of Completion by Willing to Spend")
ax[0, 1].set_ylabel("Time Passed as % of compDone")

# 7. Fourth subplot: Violin plot for Utilization for Actual data
sns.boxplot(data=actual_usage_metrics, x="nGenPerTick", y="utilization", ax=ax[1, 1])

ax[1, 1].set_title("Actual: Distribution of Utilization by nGenPerTick")
ax[1, 1].set_xlabel("nGenPerTick")
ax[1, 1].set_ylabel("Utilization")
ax[1, 1].axhline(y=1, color='grey', linewidth=2, alpha=0.5)

# 8. Fifth subplot: Violin plot for Memory Utilization for Ideal data
sns.boxplot(data=ideal_usage_metrics, x="nGenPerTick", y="mem_utilization", ax=ax[2, 0])

ax[2, 0].set_title("Ideal: Distribution of Memory Utilization by nGenPerTick")
ax[2, 0].set_xlabel("nGenPerTick")
ax[2, 0].set_ylabel("Memory Utilization")
ax[2, 0].axhline(y=1, color='grey', linewidth=2, alpha=0.5)

# 9. Sixth subplot: Violin plot for Memory Utilization for Actual data
sns.boxplot(data=actual_usage_metrics, x="nGenPerTick", y="mem_utilization", ax=ax[2, 1])

ax[2, 1].set_title("Actual: Distribution of Memory Utilization by nGenPerTick")
ax[2, 1].set_xlabel("nGenPerTick")
ax[2, 1].set_ylabel("Memory Utilization")
ax[2, 1].axhline(y=1, color='grey', linewidth=2, alpha=0.5)

# Set the same y-axis limits for each row (ideal and actual)
# For Time Passed as % of Completion (Ideal and Actual)
y_max_0 = max(ideal_procs_done["timeAsPercentage"].max(), actual_procs_done["timeAsPercentage"].max()) * 1.1
y_min_0 = -0.08 * y_max_0

# Set the limits for the first row (ax[0, 0] and ax[0, 1])
ax[0, 0].set_ylim(y_min_0, y_max_0)
ax[0, 1].set_ylim(y_min_0, y_max_0)

# Set the limits for the second row (ax[1, 0] and ax[1, 1])
ax[1, 0].set_ylim(-0.1, 1.1)
ax[1, 1].set_ylim(-0.1, 1.1)

# Set the limits for the third row (memory utilization)
ax[2, 0].set_ylim(-0.1, 1.1)
ax[2, 1].set_ylim(-0.1, 1.1)

# Adjust the layout to avoid overlap
plt.tight_layout()

# Save the figure and show it
plt.savefig('combined_res.png')
plt.show()

