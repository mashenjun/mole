import numpy as np
from dtw import dtw
import matplotlib.pyplot as plt
import matplotlib.gridspec as gridspec


def cal_sim():
    x_file = "/Users/shenjun/Workspace/data-analysis/heatmap-data/210809T140000+0800-210809T143000+0800/read_keys_tpcc.csv"
    y_file = "/Users/shenjun/Workspace/data-analysis/heatmap-data/210809T140000+0800-210809T143000+0800/read_bytes_tpcc.csv"
    z_file = "/Users/shenjun/Workspace/data-analysis/heatmap-data/210809T140000+0800-210809T143000+0800/written_keys_tpcc.csv"
    w_file = "/Users/shenjun/Workspace/data-analysis/heatmap-data/210809T140000+0800-210809T143000+0800/written_bytes_tpcc.csv"
    # x_range = col_range(x_file)
    x = np.loadtxt(x_file, delimiter=',')
    y = np.loadtxt(y_file, delimiter=',')
    z = np.loadtxt(z_file, delimiter=',')
    w = np.loadtxt(w_file, delimiter=',')

    # drop the first column, since the first column is timestamp.
    x = np.delete(x, 0, axis=1)
    y = np.delete(y, 0, axis=1)
    z = np.delete(z, 0, axis=1)
    w = np.delete(w, 0, axis=1)

    # view distribution for first 30 rows.
    f, ax = plt.subplots(10, 3, sharey=True)

    for i in range(3):
        for j in range(10):
            # do normalization row by row
            norm = normalization(x[i*10 + j])
            ax[j, i].plot(norm)
    plt.show()

    # f, ax = plt.subplots(10, 3, sharey=True)
    # for i in range(3):
    #     for j in range(10):
    #         # do normalization row by row
    #         norm = normalization(w[i*10 + j])
    #         ax[j, i].plot(norm)
    # plt.show()

    # x = normalization(x)
    # y = normalization(y)
    # z = normalization(z)
    x_std = np.std(x, axis=0)
    y_std = np.std(y, axis=0)
    z_std = np.std(z, axis=0)
    overview(x_std, "x_std", y_std, "y_std")
    overview(x_std, "x_std", z_std, "z_std")

    cal_dtw(x, y)
    cal_dtw_threeway(x, z)
    plt.show()

    return


def distance(x: np.ndarray, y: np.ndarray):
    val = np.abs(np.std(x) - np.std(y))
    return val


def overview(x: np.ndarray, x_title: str, y: np.ndarray, y_title: str):
    f, (ax_x, ax_y, ax_cmp) = plt.subplots(3, 1, sharey=False)
    ax_x.plot(x, 'g-')
    ax_x.set_title(x_title)
    ax_y.plot(y, 'r-')
    ax_y.set_title(y_title)
    x_nom = centralization(x)
    y_nom = centralization(y)
    ax_cmp.plot(x_nom, 'g-', label=x_title)
    ax_cmp.plot(y_nom, 'r-', label=y_title)
    f.legend()
    f.tight_layout()
    f.show()
    return x_nom, y_nom


def centralization(data: np.ndarray):
    return data - np.min(data)


def normalization(data: np.ndarray):
    _range = np.max(data) - np.min(data)
    return (data - np.min(data)) / _range


def col_range(file: str):
    with open(file) as f:
        ncols = len(f.readline().split(','))
    return range(1, ncols + 1)


# we need to draw a three way plot
def cal_dtw_threeway(x: np.ndarray, y: np.ndarray):
    # calculate dtw
    d, cost_matrix, acc_cost_matrix, path = dtw(x, y, dist=distance)
    print(d)

    x_std = np.std(x, axis=0)
    y_std = np.std(y, axis=0)
    nn = len(x_std)
    mm = len(y_std)
    nn1 = np.arange(nn)
    mm1 = np.arange(mm)

    fig = plt.figure()
    gs = gridspec.GridSpec(2, 2,
                           width_ratios=[1, 3],
                           height_ratios=[3, 1])
    axr = plt.subplot(gs[0])
    ax = plt.subplot(gs[1])
    axq = plt.subplot(gs[3])
    axq.plot(nn1,x_std)  # query, horizontal, bottom
    axq.set_xlabel("base")

    axr.plot(y_std, mm1)  # ref, vertical
    axr.invert_xaxis()
    axr.set_ylabel("target")

    ax.imshow(acc_cost_matrix.T, origin='lower', cmap='gray', interpolation='nearest')
    ax.plot(path[0], path[1], 'w')


def cal_dtw(x: np.ndarray, y: np.ndarray):
    # calculate dtw
    d, cost_matrix, acc_cost_matrix, path = dtw(x, y, dist=distance)
    print(d)
    plt.figure()
    plt.imshow(acc_cost_matrix.T, origin='lower', cmap='gray', interpolation='nearest')
    plt.plot(path[0], path[1], 'w')


if __name__ == "__main__":
    cal_sim()
