import os
from pathlib import Path
import numpy as np
import tabulate
from yaml import load, Loader
import pandas as pd

import prom_metrics_feature_basic
import weighted_sigmoid
import argparse


def load_feature(file: str):
    data = pd.read_csv(file)
    return data.set_index('metrics')


def load_yaml(file: str):
    f1 = open(file)  # 打开yaml文件
    return load(f1, Loader=Loader)  # 使用load方法加载


# return a df containing feature scores.
def cal_weighted_feature_score(f: pd.DataFrame, ff: dict):
    score_table_cols = ['name', 'score', 'weight']
    score_table = pd.DataFrame(columns=score_table_cols)
    for spec in ff['feature_functions']:
        name = spec['name']
        metrics_name = spec['metrics_name']
        factor = spec.get('factor', 1)
        function = spec['function']
        min_val = spec.get('min', 0)
        max_val = spec.get('max', 0)
        unit = spec.get('unit', '')
        weight = spec.get('weight', 1)
        # retrieve feature metrics
        if function == 'expit':
            gx_k = weighted_sigmoid.cal_k_for_gx(min_val, max_val)
            gx_m = (min_val - max_val) / 2
            # consider the factor
            feature_value = f.loc[metrics_name]['mean'] * factor
            # consider the unit factor
            new_feature_value = convert_unit(feature_value, unit)
            feature_score = weighted_sigmoid.gx(gx_k, gx_m, new_feature_value) * weight
            data = pd.DataFrame([[name, feature_score, weight]], columns=score_table_cols)
            score_table = score_table.append(data, ignore_index=True)
        else:
            feature_score = f.loc[metrics_name]['mean']
            data = pd.DataFrame([[name, feature_score, weight]], columns=score_table_cols)
            score_table = score_table.append(data, ignore_index=True)
    return score_table


def convert_unit(val: float, unit: str):
    # if the unit is not btyes size, do nothing
    l_unit = unit.lower()
    if l_unit == 'kb':
        return val / 1024
    elif l_unit == 'mb':
        return val / (1024 ** 2)
    elif l_unit == 'gb':
        return val / (1024 ** 3)
    elif l_unit == 'tb':
        return val / (1024 ** 4)
    else:
        return val


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="""
            prom_metrics_feature_score.py calculate feature score for target metrics""")
    parser.add_argument('-f', '--function', dest='feature_function', help='yaml contains feature function settings',
                        required=True)
    # parser.add_argument('--source', dest='source', help='csv file contain basic feature information',
    #                     required=True)
    parser.add_argument('--source_dir', dest='source_dir', help='dir contains reshaped metrics, in csv format',
                        required=True)
    parser.add_argument('-o', '--output', dest='output', help='output file stores the feature score')
    args = parser.parse_args()

    ff = load_yaml(args.feature_function)
    # f = load_feature(args.source)
    # source_dir = args.source_dir
    # output_file = args.output
    # ff = load_yaml('/Users/shenjun/Workspace/play-dtw/feature_function.yaml')
    # f = load_feature('/Users/shenjun/Workspace/data-analysis/metrics-csv/features.csv')
    # visual the table
    source_dir = args.source_dir
    f = pd.DataFrame(columns=prom_metrics_feature_basic.feature_cols)
    arr = os.listdir(source_dir)
    for i, file in enumerate(arr):
        metrics = Path(file).stem
        data = pd.read_csv(os.path.join(source_dir, file), dtype='float')
        print("extract {0} feature...".format(metrics))
        features = prom_metrics_feature_basic.extract_feature(data, metrics)
        f = f.append(features, ignore_index=True)
    f.set_index('metrics', inplace=True)
    # arr = np.empty(shape=[0, 2])
    # for spec in ff['feature_functions']:
    #     arr = np.append(arr, [[spec['min'], spec['max']]], axis=0)
    # weighted_sigmoid.visual_gxs(arr)
    # cal the score table
    score_table = cal_weighted_feature_score(f, ff)
    print(tabulate.tabulate(score_table, headers=score_table.columns))
    if args.output is not None:
        score_table.to_csv(args.output, sep=',', index=False)

