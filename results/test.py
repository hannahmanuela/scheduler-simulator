import numpy as np
import matplotlib.pyplot as plt

# Generate data from the first normal distribution
np.random.seed(0)  # For reproducibility
mu1 = 1000
sigma1 = 500
data1 = np.random.normal(mu1, sigma1, 1000)

# Generate data from the second normal distribution
mu2 = 7000
sigma2 = 2000
data2 = np.random.normal(mu2, sigma2, 1000)

# Combine the two distributions
bimodal_data = np.concatenate([data1, data2])

# Plot the histogram
plt.hist(bimodal_data, bins=30, density=True, alpha=0.6)
plt.xlabel('Value')
plt.ylabel('Frequency')
plt.title('Bimodal Distribution')
plt.show()