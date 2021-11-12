import numpy as np
import matplotlib.pyplot as plt
from scipy.special import expit


# this is our function to calculate the feature value
def gx(k, m, x):
    return expit(k*(x+m))

def ggx(k, m, x):
    return expit(k*(x+m))

# f(x) = expit(x)
# g(x) = ax+b
# f(g(x)) 的导数为 y(1-y)a
# https://blog.csdn.net/Su_Mo/article/details/79281623
def cal_k_for_gx(min_val: float, max_val: float):
    # this step just make the gx function more curvy
    rg = (max_val-min_val) / 2
    return 4 / rg


def w_sigmoid(min_val: float, max_val: float, x):
    k = cal_k_for_gx(min_val, max_val)
    m = (min_val + (max_val-min_val)/2)
    rg = max_val - min_val
    x = m + (x-min_val)/2
    return expit(k*(x-m))*2 -1


#  visual_gx sigmoid function by the given min_value and max_value
def visual_gxs(params: np.ndarray):
    f, axs = plt.subplots(len(params), 1, sharey=False, sharex=False)

    for i in range(0, len(params)):
        min_val = params[i][0]
        max_val = params[i][1]
        mean = min_val + (max_val - min_val) / 2

        gk = cal_k_for_gx(min_val, max_val)
        k = 1 / (max_val - min_val)

        print(min_val, max_val, mean, gk)
        x = np.linspace(min_val, max_val, 100)

        xx = np.linspace(min_val, max_val, 7)
        yy = gx(gk, -mean, xx)

        axs[i].plot(x, k * (x - min_val), 'b-', lw=1, alpha=0.6)
        axs[i].plot(x, gx(gk, -mean, x), 'r-', lw=1, alpha=0.6, label='logistic cdf')
        # axs[i].plot(x, w_sigmoid(min_val, max_val, x), 'g-', lw=1, alpha=0.6, label='logistic cdf')
        # axs[i].plot(x, gx(gk, -mean, x)*2, 'r-', lw=1, alpha=0.6, label='logistic cdf')


        # axs[i].plot(mean, gx(gk, -mean, mean), 'r.')
        # axs[i].text(mean, gx(gk, -mean, mean), (mean, gx(gk, -mean, mean)), ha='center', va='bottom', fontsize=10)
        #
        # axs[i].plot(mean, w_sigmoid(min_val, max_val, mean), 'g.')
        # axs[i].text(mean, w_sigmoid(min_val, max_val, mean), (mean, w_sigmoid(min_val, max_val, mean)), ha='center',
        #             va='bottom', fontsize=10)

        for a, b in zip(xx, yy):
            axs[i].hlines(b, 0, a, linestyles="--")
            axs[i].plot(a, b, 'bo')
            axs[i].text(a, b, (round(a, 3), round(b, 3)), ha='center', va='bottom', fontsize=10)

        axs[i].set_ylim(bottom=0, top=1)
        axs[i].set_xlim(left=min_val, right=max_val)
    plt.show()
    return


if __name__ == "__main__":
    params = np.array([[0.6,0.9],[1,10]])
    visual_gxs(params)