import os
from pathlib import Path
import pandas as pd
import argparse
from tsfresh.feature_extraction import extract_features, MinimalFCParameters

# cnt is the positive value count
feature_cols = ['metrics', 'sum', 'median', 'mean',
                'length', 'standard_deviation',
                'variance', 'maximum', 'minimum', 'mean_mean', 'maximum_mean', 'sum_div_cnt', 'mean_sum_div_cnt', 'maximum_sum_div_cnt']


def load_csv(file_name: str):
    return pd.read_csv(file_name, dtype='float')


# return a df contain all basic feature
def extract_feature(df: pd.DataFrame, metrics_name: str, need_summary: bool):
    table = pd.DataFrame(columns=feature_cols)
    cols = df.columns[1:]
    # print(cols)
    df['metrics'] = 1
    for col in cols:
        ddf = df[['metrics', 'timestamp', col]]
        # print(ddf.head())
        extracted_features = extract_features(ddf, column_id='metrics', column_sort="timestamp", column_value=col,
                                              default_fc_parameters=MinimalFCParameters(), disable_progressbar=True)
        index_name = metrics_name + ":" + col
        cnt = ddf[ddf[col] > 0][col].count()
        sum_div_cnt = 0 if cnt == 0 else extracted_features.iloc[0][0] / cnt
        extracted_features.insert(loc=0, column='metrics', value=index_name)
        extracted_features.insert(loc=9, column='mean_mean', value=0)
        extracted_features.insert(loc=10, column='maximum_mean', value=0)
        extracted_features.insert(loc=11, column='sum_div_cnt', value=sum_div_cnt)
        extracted_features.insert(loc=12, column='mean_sum_div_cnt', value=0)
        extracted_features.insert(loc=13, column='maximum_sum_div_cnt', value=0)
        new_cols = {x: y for x, y in zip(extracted_features.columns, table.columns)}
        # print(new_cols)
        extracted_features.rename(columns=new_cols, inplace=True)
        table = table.append(extracted_features, ignore_index=True)
    # extracted_features = extract_features(df, column_id='metrics', column_sort="timestamp")
    # calculate some summary feature
    balance_summary = pd.Series([metrics_name, 0, 0, 0,
        0, 0, 0,
        table['maximum'].max(),
        table['minimum'].min(),
        table['mean'].mean(),
        table['mean'].max(),
        0,
        table['sum_div_cnt'].mean(),
        table['sum_div_cnt'].max()],
        index= feature_cols)
    table = table.append(balance_summary, ignore_index=True)
    return table


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="""
        prom_metrics_feature_basic.py calculate basic feature for metrics""")
    parser.add_argument('-i', '--input', dest='input_dir', help='input dir contain reshaped metrics, in csv format',
                        required=True)
    parser.add_argument('-o', '--output', dest='output', help='output file store basic feature result, in csv format',
                        required=True)

    args = parser.parse_args()
    input_dir = args.input_dir
    output_file = args.output
    arr = os.listdir(input_dir)
    for i, file in enumerate(arr):
        metrics = Path(file).stem
        data = load_csv(os.path.join(input_dir, file))
        print("extract {0} feature...".format(metrics))
        features = extract_feature(data, metrics, True)
        # write feature row by row
        if i == 0:
            features.to_csv(output_file, sep=',', index=False)
        else:
            features.to_csv(output_file, sep=',', index=False, header=False, mode='a')
