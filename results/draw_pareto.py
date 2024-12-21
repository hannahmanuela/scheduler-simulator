import numpy as np
import matplotlib.pyplot as plt

# Parameters for the Pareto distribution
alpha = 25.0    # Shape parameter (alpha > 0)
x_m = 50.0      # Scale parameter (x_m > 0)

# Generate random samples from the Pareto distribution
sample_size = 1000
pareto_samples = (np.random.pareto(alpha, sample_size) + 1) * x_m

# Plotting the Pareto distribution
plt.figure(figsize=(8, 6))
plt.hist(pareto_samples, bins=50, density=True, alpha=0.6, color='g')

# Plot the theoretical PDF of the Pareto distribution
x = np.linspace(x_m, pareto_samples.max(), 1000)
pdf = (alpha * x_m**alpha) / (x**(alpha + 1))
plt.plot(x, pdf, 'r-', lw=2, label=f'Pareto PDF (a={alpha}, xm={x_m})')

# Labels and title
plt.title('Pareto Distribution (a = 2, xm = 1)')
plt.xlabel('x')
plt.ylabel('Density')
plt.legend()

# Show the plot
plt.grid(True)
plt.show()
