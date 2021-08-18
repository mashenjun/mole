import numpy as np
from dtw import dtw
import matplotlib.pyplot as plt
import sys


def peek(x_file: str):
    x = np.loadtxt(x_file, delimiter=',')

    # drop the first column, since the first column is timestamp.
    x = np.delete(x, 0, axis=1)

    # view distribution for first 30 rows.
    f, ax = plt.subplots(10, 2, sharey=True)

    for i in range(2):
        for j in range(10):
            # do normalization row by row
            norm = normalization(x[i*10 + j])
            ax[j, i].bar(norm)
    plt.show()
    return


def centralization(data: np.ndarray):
    return data - np.min(data)


def normalization(data: np.ndarray):
    _range = np.max(data) - np.min(data)
    return (data - np.min(data)) / _range


if __name__ == "__main__":
    if len(sys.argv) != 2:
        raise ValueError(__file__+' <heatmap.csv>')
    file = sys.argv[1]
    peek(file)
