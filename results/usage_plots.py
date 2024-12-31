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
actual_procs_done = pd.read_csv("procs_done.txt", index_col=None, names=["nGenPerTick", "willingToSpend", "timePassed", "compDone"])

hermod_usage_metrics = pd.read_csv("hermod_usage.txt", index_col=None, names=["nGenPerTick", "tick", "machineId", "ticksLeftOver", "memFree"])
hermod_procs_done = pd.read_csv("hermod_procs_done.txt", index_col=None, names=["nGenPerTick", "willingToSpend", "timePassed", "compDone"])

# 1. Compute timeAsPercentage for both ideal and actual data
ideal_procs_done["timeAsPercentage"] = (ideal_procs_done["timePassed"] / ideal_procs_done["compDone"]) * 100
actual_procs_done["timeAsPercentage"] = (actual_procs_done["timePassed"] / actual_procs_done["compDone"]) * 100
hermod_procs_done["timeAsPercentage"] = (hermod_procs_done["timePassed"] / hermod_procs_done["compDone"]) * 100

# 2. Compute utilization for both ideal and actual data
ideal_usage_metrics["utilization"] = (numMachines * coresPerMachine - ideal_usage_metrics["ticksLeftOver"]) / (numMachines * coresPerMachine)
actual_usage_metrics["utilization"] = (coresPerMachine - actual_usage_metrics["ticksLeftOver"]) / (coresPerMachine)
hermod_usage_metrics["utilization"] = (coresPerMachine - hermod_usage_metrics["ticksLeftOver"]) / (coresPerMachine)

# 3. Compute memory utilization (1 - memFree / totalMemory)
ideal_usage_metrics["mem_utilization"] = 1 - (ideal_usage_metrics["memFree"] / (numMachines * totalMemoryPerMachine))
actual_usage_metrics["mem_utilization"] = 1 - (actual_usage_metrics["memFree"] / totalMemoryPerMachine)
hermod_usage_metrics["mem_utilization"] = 1 - (hermod_usage_metrics["memFree"] / totalMemoryPerMachine)

print("num ideal finished: ", len(ideal_procs_done))
print("num actual finished: ", len(actual_procs_done))
print("num hermod finished: ", len(hermod_procs_done))


# Create the figure with nine subplots (3 rows, 3 columns)
fig, ax = plt.subplots(3, 3, figsize=(15, 9), sharex=True)

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

# 6. Third subplot: Violin plot for Memory Utilization for Ideal data
sns.boxplot(data=ideal_usage_metrics, x="nGenPerTick", y="mem_utilization", ax=ax[2, 0])

ax[2, 0].set_title("Ideal: Distribution of Memory Utilization by nGenPerTick")
ax[2, 0].set_xlabel("nGenPerTick")
ax[2, 0].set_ylabel("Memory Utilization")
ax[2, 0].axhline(y=1, color='grey', linewidth=2, alpha=0.5)

# 7. Fourth subplot: Strip plot for Time Passed as % of Completion for Actual data
sns.stripplot(data=actual_procs_done, x="nGenPerTick", y="timeAsPercentage", hue="willingToSpend", 
              jitter=True, palette=high_contrast_palette, ax=ax[0, 1])

ax[0, 1].set_title("Actual: Time Passed as % of Completion by Willing to Spend")
ax[0, 1].set_ylabel("Time Passed as % of compDone")

# 8. Fifth subplot: Violin plot for Utilization for Actual data
sns.boxplot(data=actual_usage_metrics, x="nGenPerTick", y="utilization", ax=ax[1, 1])

ax[1, 1].set_title("Actual: Distribution of Utilization by nGenPerTick")
ax[1, 1].set_xlabel("nGenPerTick")
ax[1, 1].set_ylabel("Utilization")
ax[1, 1].axhline(y=1, color='grey', linewidth=2, alpha=0.5)

# 9. Sixth subplot: Violin plot for Memory Utilization for Actual data
sns.boxplot(data=actual_usage_metrics, x="nGenPerTick", y="mem_utilization", ax=ax[2, 1])

ax[2, 1].set_title("Actual: Distribution of Memory Utilization by nGenPerTick")
ax[2, 1].set_xlabel("nGenPerTick")
ax[2, 1].set_ylabel("Memory Utilization")
ax[2, 1].axhline(y=1, color='grey', linewidth=2, alpha=0.5)

# 10. Seventh subplot: Strip plot for Time Passed as % of Completion for Hermod data
sns.stripplot(data=hermod_procs_done, x="nGenPerTick", y="timeAsPercentage", hue="willingToSpend", 
              jitter=True, palette=high_contrast_palette, ax=ax[0, 2])

ax[0, 2].set_title("Hermod: Time Passed as % of Completion by Willing to Spend")
ax[0, 2].set_ylabel("Time Passed as % of compDone")

# 11. Eighth subplot: Violin plot for Utilization for Hermod data
sns.boxplot(data=hermod_usage_metrics, x="nGenPerTick", y="utilization", ax=ax[1, 2])

ax[1, 2].set_title("Hermod: Distribution of Utilization by nGenPerTick")
ax[1, 2].set_xlabel("nGenPerTick")
ax[1, 2].set_ylabel("Utilization")
ax[1, 2].axhline(y=1, color='grey', linewidth=2, alpha=0.5)

# 12. Ninth subplot: Violin plot for Memory Utilization for Hermod data
sns.boxplot(data=hermod_usage_metrics, x="nGenPerTick", y="mem_utilization", ax=ax[2, 2])

ax[2, 2].set_title("Hermod: Distribution of Memory Utilization by nGenPerTick")
ax[2, 2].set_xlabel("nGenPerTick")
ax[2, 2].set_ylabel("Memory Utilization")
ax[2, 2].axhline(y=1, color='grey', linewidth=2, alpha=0.5)

# Set the same y-axis limits for each row (ideal, actual, hermod)
# For Time Passed as % of Completion (Ideal, Actual, and Hermod)
y_max_0 = max(ideal_procs_done["timeAsPercentage"].max(), actual_procs_done["timeAsPercentage"].max(), hermod_procs_done["timeAsPercentage"].max()) * 1.1
y_min_0 = -0.08 * y_max_0

# Set the limits for the first row (ax[0, 0], ax[0, 1], ax[0, 2])
ax[0, 0].set_ylim(y_min_0, y_max_0)
ax[0, 1].set_ylim(y_min_0, y_max_0)
ax[0, 2].set_ylim(y_min_0, y_max_0)

# Set the limits for the second row (ax[1, 0], ax[1, 1], ax[1, 2])
ax[1, 0].set_ylim(-0.1, 1.1)
ax[1, 1].set_ylim(-0.1, 1.1)
ax[1, 2].set_ylim(-0.1, 1.1)

# Set the limits for the third row (memory utilization)
ax[2, 0].set_ylim(-0.1, 1.1)
ax[2, 1].set_ylim(-0.1, 1.1)
ax[2, 2].set_ylim(-0.1, 1.1)

# Adjust the layout to avoid overlap
plt.tight_layout()

# Save the figure and show it
plt.savefig('combined_res.png')
plt.show()