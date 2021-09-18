import os
from pathlib import Path
import numpy as np
import tabulate
from yaml import load, Loader
import pandas as pd

import prom_metrics_feature_basic
import weighted_sigmoid
import argparse
import re

score_table_cols = ['name', 'score', 'weight', 'distance_function']

def load_feature(file: str):
    data = pd.read_csv(file)
    return data.set_index('metrics')


def load_yaml(file: str):
    f1 = open(file)  # 打开yaml文件
    return load(f1, Loader=Loader)  # 使用load方法加载


# return a df containing feature scores.
def cal_weighted_feature_score(f: pd.DataFrame, ff: dict):
    # distance_function control how to cal distance
    score_table = pd.DataFrame(columns=score_table_cols)
    for spec in ff['feature_functions']:
        name = spec['name']
        metrics_name = spec['metrics_name']
        feature_name = spec.get('feature_name', 'mean')
        factor = spec.get('factor', 1)
        function = spec['function']
        min_val = spec.get('min', 0)
        max_val = spec.get('max', 0)
        unit = spec.get('unit', '')
        weight = spec.get('weight', 1)
        distance_function = spec.get('distance_function', 'delta')
        need_reverse = False
        if min_val > max_val:
            need_reverse = True
            min_val, max_val = max_val, min_val
        # check if metrics name exist, with regex match logic
        if metrics_name not in list(f.index):
            metrics_name = metrics_name.replace("__IP__", "(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]):\d+")
            ok = False
            for index_name in list(f.index):
                if re.match(metrics_name, index_name):
                    metrics_name = index_name
                    ok = True
                    break
            if not ok:
                raise ValueError(metrics_name + " is not found in feature df")
        # retrieve feature metrics
        feature_score = 0
        if function == 'expit':
            gx_k = weighted_sigmoid.cal_k_for_gx(min_val, max_val)
            gx_m = (min_val - max_val) / 2
            # consider the factor
            feature_value = f.loc[metrics_name][feature_name] * factor
            # consider the unit factor
            new_feature_value = convert_unit(feature_value, unit)
            feature_score = weighted_sigmoid.gx(gx_k, gx_m, new_feature_value)
            # print(feature_name, feature_value, feature_score)
            if need_reverse:
                feature_score = 1 - feature_score
        elif function == 'balance':
            a = f.loc[metrics_name]['maximum_mean']
            b = f.loc[metrics_name]['mean_mean']
            # if mean_mean is zero, all instance has zero value on this metrics
            # thus set feature_score to zero directly
            feature_score = 0 if b == 0 else min((a - b) / b, 1)
        else:
            feature_score = f.loc[metrics_name][feature_name]

        data = pd.DataFrame([[name, feature_score, weight, distance_function]], columns=score_table_cols)
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


def cal_key(s: pd.Series):
    return s.apply(lambda x: x if x > 1 else 1-x)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="""
            prom_metrics_feature_score.py calculate feature score for target metrics""")
    parser.add_argument('-f', '--function', dest='feature_function', help='yaml contains feature function settings',
                        required=True)
    parser.add_argument('-i', '--input', dest='input_dir', help='input dir contains reshaped metrics, in csv format',
                        required=True)
    parser.add_argument('-o', '--output', dest='output', help='output file stores the feature score')
    args = parser.parse_args()

    ff = load_yaml(args.feature_function)
    need_summary_set = set()
    for spec in ff['feature_functions']:
        if spec['function'] == 'balance' or spec.get('cal_summary', False):
            need_summary_set.add(spec['metrics_name'])
    # visual the table
    input_dir = args.input_dir
    f = pd.DataFrame(columns=prom_metrics_feature_basic.feature_cols)
    arr = os.listdir(input_dir)
    for i, file in enumerate(arr):
        metrics = Path(file).stem
        data = pd.read_csv(os.path.join(input_dir, file), dtype='float')
        data.fillna(0, inplace=True)
        print("extract {0} feature...".format(metrics))
        need_summary = metrics in need_summary_set
        features = prom_metrics_feature_basic.extract_feature(data, metrics, need_summary)
        f = f.append(features, ignore_index=True)
    f.set_index('metrics', inplace=True)
    # arr = np.empty(shape=[0, 2])
    # for spec in ff['feature_functions']:
    #     arr = np.append(arr, [[spec['min'], spec['max']]], axis=0)
    # weighted_sigmoid.visual_gxs(arr)
    # cal the score table
    score_table = cal_weighted_feature_score(f, ff)
    if args.output is not None:
        score_table.to_csv(args.output, sep=',', index=False)
    # polish the score_table to get a more viewable result
    score_table.sort_values(by='score', ascending=True, ignore_index=True, inplace=True, key=cal_key)

    print_columns = ["weight", "score", "name"]
    print(tabulate.tabulate(score_table[print_columns], headers=print_columns, floatfmt=".3f"))


