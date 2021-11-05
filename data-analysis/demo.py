import matplotlib.pyplot as plt
import pandas as pd
import numpy as np
from statsmodels.tsa.stattools import adfuller


def subplot(cluster: str, metrics: str, ax):
    df = pd.read_csv('/Users/shenjun/Workspace/xiaomi-data/{}/reshaped/{}.csv'.format(cluster, metrics))
    df.fillna(0, inplace=True)
    data = df[df.columns[1:]]
    col_avg = data.mean(axis=0)
    data['avg'] = data.apply(lambda x: x.mean(), axis=1)
    data['upper_bound'] = data.apply(lambda x: x.max(), axis=1)
    data['area'] = data.apply(lambda x: x['upper_bound'] - x['avg'], axis=1)

    ax.fill_between(data.index, data['upper_bound'], data['avg'], alpha=0.2)
    ax.set_title(metrics)
    return 'width {:.5f}, area_ratio {:.5f}'.format(data['area'].sum(axis=0) / len(data),data['area'].sum(axis=0) / data['upper_bound'].sum(axis=0))

    # ax.text(0, 0, 'total area {:.3f}, width {:.3f}, area_ratio {:.3f}'.format(data['area'].sum(axis=0),
    #                                                                                   data['area'].sum(axis=0) / len(
    #                                                                                       data),
    #                                                                                   data['area'].sum(axis=0) / data[
    #                                                                                       'upper_bound'].sum(axis=0)))

def draw(cluster: str):
    fig, axes = plt.subplots(nrows=3, ncols=4)
    a = subplot('cluster0', 'tikv_p99_rt:copr:by_instance', axes[0,0])
    b = subplot('cluster1', 'tikv_p99_rt:copr:by_instance', axes[0,0])
    axes[0,0].annotate('{}\n{}'.format(a,b), xy=(0.1,0.7), xycoords='axes fraction')

    a = subplot('cluster0', 'tikv_p99_rt:write:by_instance', axes[1,0])
    b = subplot('cluster1', 'tikv_p99_rt:write:by_instance', axes[1,0])
    axes[1,0].annotate('{}\n{}'.format(a,b), xy=(0.1,0.7), xycoords='axes fraction')

    a = subplot('cluster0', 'tikv_p99_rt:get:by_instance', axes[2,0])
    b = subplot('cluster1', 'tikv_p99_rt:get:by_instance', axes[2,0])
    axes[2,0].annotate('{}\n{}'.format(a,b), xy=(0.1,0.7), xycoords='axes fraction')

    a = subplot('cluster0', 'tikv_qps:copr:by_instance', axes[0,1])
    b = subplot('cluster1', 'tikv_qps:copr:by_instance', axes[0,1])
    axes[0,1].annotate('{}\n{}'.format(a,b), xy=(0.1,0.7), xycoords='axes fraction')

    a = subplot('cluster0', 'tikv_qps:write:by_instance', axes[1,1])
    b = subplot('cluster1', 'tikv_qps:write:by_instance', axes[1,1])
    axes[1,1].annotate('{}\n{}'.format(a,b), xy=(0.1,0.7), xycoords='axes fraction')

    a = subplot('cluster0', 'tikv_qps:get:by_instance', axes[2,1])
    b = subplot('cluster1', 'tikv_qps:get:by_instance', axes[2,1])
    axes[2,1].annotate('{}\n{}'.format(a,b), xy=(0.1,0.7), xycoords='axes fraction')

    a = subplot('cluster0', 'tikv_avg_rt:copr:by_instance', axes[0,2])
    b = subplot('cluster1', 'tikv_avg_rt:copr:by_instance', axes[0,2])
    axes[0,2].annotate('{}\n{}'.format(a,b), xy=(0.1,0.7), xycoords='axes fraction')

    a = subplot('cluster0', 'tikv_avg_rt:write:by_instance', axes[1,2])
    b = subplot('cluster1', 'tikv_avg_rt:write:by_instance', axes[1,2])
    axes[1,2].annotate('{}\n{}'.format(a,b), xy=(0.1,0.7), xycoords='axes fraction')

    a = subplot('cluster0', 'tikv_avg_rt:get:by_instance', axes[2,2])
    b = subplot('cluster1', 'tikv_avg_rt:get:by_instance', axes[2,2])
    axes[2,2].annotate('{}\n{}'.format(a,b), xy=(0.1,0.7), xycoords='axes fraction')

    a = subplot('cluster0', 'tidb_p99_rt:by_instance', axes[0,3])
    b = subplot('cluster1', 'tidb_p99_rt:by_instance', axes[0,3])
    axes[0,3].annotate('{}\n{}'.format(a,b), xy=(0.1,0.7), xycoords='axes fraction')

    a = subplot('cluster0', 'tidb_qps:by_instance', axes[1,3])
    b = subplot('cluster1', 'tidb_qps:by_instance', axes[1,3])
    axes[1,3].annotate('{}\n{}'.format(a,b), xy=(0.1,0.7), xycoords='axes fraction')

    a = subplot('cluster0', 'tidb_avg_rt:by_instance', axes[2,3])
    b = subplot('cluster1', 'tidb_avg_rt:by_instance', axes[2,3])
    axes[2,3].annotate('{}\n{}'.format(a,b), xy=(0.1,0.7), xycoords='axes fraction')

    plt.show()


    # df1 = pd.read_csv('/Users/shenjun/Workspace/xiaomi-data/{}/reshaped/tikv_p99_rt:copr:by_instance.csv'.format(cluster))
    # df1.fillna(0, inplace=True)
    # data = df1[df1.columns[1:]]
    # col_avg = data.mean(axis=0)
    # data['avg'] = data.apply(lambda x: x.mean(), axis=1)
    # data['upper_bound'] = data.apply(lambda x: x.max(), axis=1)
    # data['area'] = data.apply(lambda x: x['upper_bound'] - x['avg'], axis=1)
    # # for i in range(1,len(df1.columns)):
    # #     res = adfuller(df1.iloc[:, i])
    # #     print('{} is {}'.format(df1.columns[i], res[1] < 0.05))
    # #     print(res)
    # # df2 = pd.read_csv('/Users/shenjun/Workspace/xiaomi-data/cluster1/reshaped/tikv_qps:copr:total.csv')
    # # result2 = adfuller(df2['agg_val'])
    # # print(result2)
    #
    # df2 = pd.read_csv('/Users/shenjun/Workspace/xiaomi-data/{}/reshaped/tikv_p99_rt:write:by_instance.csv'.format(cluster))
    # df2.fillna(0, inplace=True)
    # data2 = df2[df2.columns[1:]]
    # data2['avg'] = data2.apply(lambda x: x.mean(), axis=1)
    # data2['upper_bound'] = data2.apply(lambda x: x.max(), axis=1)
    # data2['area'] = data2.apply(lambda x: x['upper_bound'] - x['avg'], axis=1)
    #
    # df3 = pd.read_csv('/Users/shenjun/Workspace/xiaomi-data/{}/reshaped/tikv_qps:copr:by_instance.csv'.format(cluster))
    # df3.fillna(0, inplace=True)
    # data3 = df3[df3.columns[1:]]
    # data3['avg'] = data3.apply(lambda x: x.mean(), axis=1)
    # data3['upper_bound'] = data3.apply(lambda x: x.max(), axis=1)
    # data3['area'] = data3.apply(lambda x: x['upper_bound'] - x['avg'], axis=1)
    #
    # df4 = pd.read_csv('/Users/shenjun/Workspace/xiaomi-data/{}/reshaped/tikv_qps:write:by_instance.csv'.format(cluster))
    # df4.fillna(0, inplace=True)
    # data4 = df4[df4.columns[1:]]
    # data4['avg'] = data4.apply(lambda x: x.mean(), axis=1)
    # data4['upper_bound'] = data4.apply(lambda x: x.max(), axis=1)
    # data4['area'] = data4.apply(lambda x: x['upper_bound'] - x['avg'], axis=1)
    #
    #
    # axes[0, 0].fill_between(data.index, data['upper_bound'], data['avg'], alpha=0.2)
    # axes[0, 0].set_title('copr_p99_rt')
    # axes[0, 0].text(0, 0, 'total area {:.3f}, width {:.3f}, area_ratio {:.3f}'.format(data['area'].sum(axis=0),
    #                                                                                   data['area'].sum(axis=0) / len(
    #                                                                                       data),
    #                                                                                   data['area'].sum(axis=0) / data[
    #                                                                                       'upper_bound'].sum(axis=0)))
    #
    # axes[1, 0].fill_between(data2.index, data2['upper_bound'], data2['avg'], alpha=0.2)
    # axes[1, 0].set_title('write_p99_rt')
    # axes[1, 0].text(0, 0, 'total area {:.3f}, width {:.3f}, area_ratio {:.3f}'.format(data2['area'].sum(axis=0),
    #                                                                                   data2['area'].sum(axis=0) / len(
    #                                                                                       data2),
    #                                                                                   data2['area'].sum(axis=0) / data2[
    #                                                                                       'upper_bound'].sum(axis=0)))
    #
    # axes[0, 1].fill_between(data.index, data3['upper_bound'], data3['avg'], alpha=0.2)
    # axes[0, 1].set_title('copr_qps')
    # axes[0, 1].text(0, 0, 'total area {:.3f}, width {:.3f}, area_ratio {:.3f}'.format(data3['area'].sum(axis=0),
    #                                                                                   data3['area'].sum(axis=0) / len(
    #                                                                                       data3),
    #                                                                                   data3['area'].sum(axis=0) / data3[
    #                                                                                       'upper_bound'].sum(axis=0)))
    #
    # axes[1, 1].fill_between(data2.index, data4['upper_bound'], data4['avg'], alpha=0.2)
    # axes[1, 1].set_title('write_qps')
    # axes[1, 1].text(0, 0, 'total area {:.3f}, width {:.3f}, area_ratio {:.3f}'.format(data4['area'].sum(axis=0),
    #                                                                                   data4['area'].sum(axis=0) / len(
    #                                                                                       data4),
    #                                                                                   data4['area'].sum(axis=0) / data4[
    #                                                                                       'upper_bound'].sum(axis=0)))
    # plt.show()

if __name__ == '__main__':
    draw('')



