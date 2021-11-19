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

print_columns = ["weight", "score", "name"]
verbose_columns = ["weight", "score", "value", "detail", "name"]
score_table_cols = ['name', 'score', 'weight', 'distance_function', 'value', 'detail', 'valid', 'skip_sort']

class color:
   PURPLE = '\033[95m'
   CYAN = '\033[96m'
   DARKCYAN = '\033[36m'
   BLUE = '\033[94m'
   GREEN = '\033[92m'
   YELLOW = '\033[93m'
   RED = '\033[91m'
   BOLD = '\033[1m'
   UNDERLINE = '\033[4m'
   END = '\033[0m'


def load_feature(file: str):
    data = pd.read_csv(file)
    return data.set_index('metrics')


def load_yaml(file: str):
    f1 = open(file)  # 打开yaml文件
    return load(f1, Loader=Loader)  # 使用load方法加载


def can_be_int(num):
    if isinstance(num, int):
        return True
    if isinstance(num, float):
        return num.is_integer()
    return False


def clear_tailing_decimal(val):
    if can_be_int(val):
        return "{:.0f}".format(val)
    if can_be_int(val*10):
        return "{:.1f}".format(val)
    if can_be_int(val*100):
        return "{:.2f}".format(val)
    return "{:.3f}".format(val)


# value should already normalized by the unit
def format_value_with_unit(val, unit: str):
    if unit == '':
        return "{0}".format(clear_tailing_decimal(val))
    elif unit == '%':
        if can_be_int(val*100):
            return "{0:.0%}".format(val)
        if can_be_int(val*1000):
            return "{0:.1%}".format(val)
        return "{0:.2%}".format(val)
    else:
        return "{0}{1}".format(clear_tailing_decimal(val), unit)


def cal_score_factor(feature_table: pd.DataFrame, spec: dict):
    if 'score_factor' not in spec:
        return 0.0, False

    if 'percentage' in spec['score_factor']:
        pct = spec['score_factor']['percentage']
        a, b = 0.0, 0.0
        for name in pct["numerator"]:
            words = name.split("@")
            a += feature_table.loc[words[0]][words[1]]
        for name in pct["denominator"]:
            words = name.split("@")
            b += feature_table.loc[words[0]][words[1]]
        if b == 0.0:
            return 1.0, True
        return a/b, True
    return 0.0, False


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
        upper_bound = spec.get('upper_bound', 1)
        value_reverse = spec.get('value_reverse', False)
        skip_sort = spec.get('skip_sort', False)
        score_factor, has_score_factor = cal_score_factor(f, spec)
        score_reverse = False
        if min_val > max_val:
            score_reverse = True
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
        # retrieve feature metrics and consider the factor
        feature_value = f.loc[metrics_name][feature_name] * factor
        valid = f.loc[metrics_name]['length'] > 0
        if function == 'expit':
            gx_k = weighted_sigmoid.cal_k_for_gx(min_val, max_val)
            gx_m = -(min_val + max_val) / 2
            # consider the unit factor and upper bound
            feature_value_unit = convert_unit_upper(feature_value, unit)
            feature_value_unit_ub = nrm_by_upper_bound(feature_value_unit, upper_bound)
            feature_score = weighted_sigmoid.gx(gx_k, gx_m, feature_value_unit_ub)
            if score_reverse:
                feature_score = 1 - feature_score
            format_value = format_value_with_unit(feature_value_unit, unit)
            if has_score_factor:
                feature_score = feature_score * score_factor
                format_value = "{}*{}".format(format_value, format_value_with_unit(score_factor, ''))
            if upper_bound > 1:
                detail = "expit({},{},{}),{},{}".format(format_value_with_unit(spec.get('min', 0), unit),
                                                        format_value_with_unit(spec.get('max', 0), unit),
                                                        upper_bound, feature_name, distance_function)
            else:
                detail = "expit({},{}),{},{}".format(format_value_with_unit(spec.get('min', 0), unit),
                                                     format_value_with_unit(spec.get('max', 0), unit),
                                                     feature_name, distance_function)
        elif function == 'balance':
            a_unit = convert_unit_upper(f.loc[metrics_name]['maximum_mean'], unit)
            b_unit = convert_unit_upper(f.loc[metrics_name]['mean_mean'], unit)
            a = nrm_by_upper_bound(a_unit, upper_bound)
            b = nrm_by_upper_bound(b_unit, upper_bound)
            # if mean_mean is zero, all instance has zero value on this metrics
            # thus set feature_score to zero directly
            feature_score = 0 if b == 0 else min((a - b) / b, 1)
            format_value = "{0},{1}".format(format_value_with_unit(a_unit, unit), format_value_with_unit(b_unit, unit))
            if has_score_factor:
                feature_score = feature_score * score_factor
                format_value = "{}*{}".format(format_value, format_value_with_unit(score_factor, ''))
            detail = "{},{}".format(function, distance_function)
        else:
            if value_reverse:
                feature_value = 1 - feature_value
            feature_value_unit = convert_unit_upper(feature_value, unit)
            feature_score = nrm_by_upper_bound(feature_value_unit, upper_bound)
            format_value = format_value_with_unit(feature_value_unit, unit)
            if has_score_factor:
                feature_score = feature_score * score_factor
                format_value = "{}*{}".format(format_value, format_value_with_unit(score_factor, ''))
            detail = "{},{},{}".format(function, feature_name, distance_function)

        format_value = '{0}(invalid)'.format(format_value) if valid is False else format_value
        data = pd.DataFrame([[name, feature_score, weight, distance_function, format_value, detail, valid, skip_sort]], columns=score_table_cols)
        score_table = score_table.append(data, ignore_index=True)
    return score_table


def convert_unit_upper(val: float, unit: str):
    # if the unit is not bytes size, do nothing
    l_unit = unit.lower()
    result = val
    # bytes to other unit
    if l_unit == 'kb':
        result = val / 1024
    elif l_unit == 'mb':
        result = val / (1024 ** 2)
    elif l_unit == 'gb':
        result = val / (1024 ** 3)
    elif l_unit == 'tb':
        result = val / (1024 ** 4)
    # seconds to other unit
    elif l_unit == 'ms':
        result = val * 1000
    return result


# val and upper_bound must in same unit
def nrm_by_upper_bound(val: float, upper_bound: float):
    result = val
    if upper_bound > 1.0:
        result = val / upper_bound
    return result


def cal_key(s):
    return s.apply(lambda x: 1+x['score'] if x['skip_sort'] else x)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="""
            prom_metrics_feature_score.py calculate feature score for target metrics""")
    parser.add_argument('-f', '--function', dest='feature_function', help='yaml contains feature function settings',
                        required=True)
    parser.add_argument('-i', '--input', dest='input_dir', help='input dir contains reshaped metrics, in csv format',
                        required=True)
    parser.add_argument('-o', '--output', dest='output', help='output file stores the feature score')
    parser.add_argument('-vv', '--verbose', dest='verbose', type=bool, default=False, help='if verbose is set, show detail in result table')
    args = parser.parse_args()
    input_dir = args.input_dir
    verbose = args.verbose

    feature_function_spec = load_yaml(args.feature_function)
    meta = load_yaml(os.path.join(input_dir, "meta.yaml"))
    need_summary_set = set()
    # use tikv_instance_cnt to replace some factor
    for spec in feature_function_spec['feature_functions']:
        if spec['function'] == 'balance' or spec.get('cal_summary', False):
            need_summary_set.add(spec['metrics_name'])
        if spec.get('factor', 1) == -1:
            spec['factor'] = meta['tikv_instance_cnt']
    extracted_feature = pd.DataFrame(columns=prom_metrics_feature_basic.feature_cols)
    arr = os.listdir(input_dir)
    for i, file in enumerate(arr):
        if Path(file).suffix == '.yaml' or Path(file).suffix == '.yml':
            continue
        metrics = Path(file).stem
        data = pd.read_csv(os.path.join(input_dir, file), dtype='float')
        # nan occurs when metrics divided by a no existed value
        data.fillna(0, inplace=True)
        # inf occurs when metrics divided by a zero value
        data.replace(to_replace=np.inf, value=0, inplace=True)
        print("extract {0} feature...".format(metrics))
        need_summary = metrics in need_summary_set
        features = prom_metrics_feature_basic.extract_feature(data, metrics, need_summary)
        extracted_feature = extracted_feature.append(features, ignore_index=True)
    extracted_feature.set_index('metrics', inplace=True)
    # arr = np.empty(shape=[0, 2])
    # for spec in ff['feature_functions']:
    #     arr = np.append(arr, [[spec['min'], spec['max']]], axis=0)
    # weighted_sigmoid.visual_gxs(arr)
    # cal the score table
    score_table = cal_weighted_feature_score(extracted_feature, feature_function_spec)
    if args.output is not None:
        score_table.to_csv(args.output, sep=',', index=False)
    # polish the score_table to get a more viewable result
    score_table['val_to_sort'] = score_table.apply(lambda x: 0 - x['score'] if x['skip_sort'] is True else x['score'], axis=1)
    score_table.sort_values(by='val_to_sort', ascending=False, ignore_index=True, inplace=True)
    score_table['name'] = score_table.apply(lambda x: color.UNDERLINE + x['name'] + color.END if x['skip_sort'] is True else x['name'], axis=1)
    if verbose:
        print(tabulate.tabulate(score_table[verbose_columns], headers=verbose_columns, floatfmt=".3f"))
    else:
        print(tabulate.tabulate(score_table[print_columns], headers=print_columns, floatfmt=".3f"))


