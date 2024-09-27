import numpy as np
import pandas as pd
from datetime import datetime

from scipy.optimize import curve_fit
from astropy import convolution, io, table

import json


def load_file(file_path):
    data_frame = None
    if file_path.endswith(".lc"):
        with io.fits.open(file_path) as fits_file:
            fits_file.verify("fix")
            data_frame = pd.DataFrame(fits_file[1].data)

    elif file_path.endswith(".csv"):
        data_frame = pd.read_csv(file_path)

    elif file_path.endswith(".fits"):
        with io.fits.open(file_path) as fits_file:
            fits_file.verify("fix")
            data_frame = table.Table(fits_file[1].data)

    if isinstance(data_frame["TIME"][0], str):
        data_frame["TIME"] = data_frame["TIME"].apply(to_MET)

    rebin_data = rebin_data_func(data_frame, 60)
    rebin_data["RATE"] = convolution.convolve(
        rebin_data["RATE"], convolution.Box1DKernel(10)
    )
    return identify_flare(rebin_data)


def to_MET(time_string):
    formats = [
        "%d-%m-%Y %H:%M:%S.%f",
        "%Y-%m-%d %H:%M:%S.%f",
        "%m-%d-%Y %H:%M:%S.%f",
    ]

    for fmt in formats:
        try:
            timestamp = datetime.strptime(time_string, fmt).timestamp()
            return timestamp - datetime(2017, 1, 1).timestamp()
        except:
            continue

    return np.float(time_string)


def rebin_data_func(data, time_step):
    result = {"TIME": [], "RATE": []}
    for i in range(0, len(data), time_step):
        result["TIME"].append(data["TIME"][i])
        sum_rates = (
            sum(data["RATE"][i : i + time_step])
            if i + time_step < len(data)
            else sum(data["RATE"][i:])
        )
        result["RATE"].append(sum_rates / time_step)

    return pd.DataFrame(result)


def poly_fit(x, a, b, c):
    return ((-1 * b) / (2 * a)) - np.sqrt(((x - c) / a) + (b * b / 4 * a * a))


def decay_fit(x, A, alpha):
    return (x / abs(A)) ** (-alpha)


def estimate_flare_start_points(rate, time, background, peak_time):
    param, _ = curve_fit(
        poly_fit,
        rate,
        np.array(time) - peak_time,
        [-1, 1, 1],
        bounds=([-np.inf, -np.inf, -np.inf], [-1, -0.1, np.inf]),
        maxfev=50000,
    )
    return (
        -(param[1] / 2 * param[0])
        - np.sqrt(
            ((background - param[2]) / param[0])
            + (param[1] * param[1] / 4 * param[0] * param[0])
        )
        + peak_time
    )


def estimate_decay_end(rate, time, background, peak_time):
    param, _ = curve_fit(
        decay_fit,
        rate,
        np.array(time) - peak_time,
        [1, 1],
        bounds=([0.0001, -np.inf], np.inf),
        maxfev=50000,
    )
    return ((background + 0.47) / param[0]) ** (-param[1]) + peak_time


def identify_flare(data_frame):
    (
        flare_start_times,
        flare_peak_times,
        flare_end_times,
        flare_rates,
        flare_start_points,
        flare_background_levels,
        flare_peak_values,
        flare_type,
    ) = ([] for _ in range(8))

    lc_index = 0
    background_level = data_frame["RATE"][0]

    while lc_index < (len(data_frame) - 4):
        if (
            (data_frame["RATE"][lc_index] < data_frame["RATE"][lc_index + 1])
            and (data_frame["RATE"][lc_index + 1] < data_frame["RATE"][lc_index + 2])
            and (data_frame["RATE"][lc_index + 2] < data_frame["RATE"][lc_index + 3])
            and (data_frame["RATE"][lc_index + 3] < data_frame["RATE"][lc_index + 4])
            and (data_frame["RATE"][lc_index + 4] / data_frame["RATE"][lc_index] > 1.03)
        ):
            flare_start_times.append(data_frame["TIME"][lc_index])
            flare_rates.append(data_frame["RATE"][lc_index])
            next_index = lc_index + 5

            for j in range(lc_index + 4, len(data_frame) - 3):
                if (
                    (data_frame["RATE"][j] > data_frame["RATE"][j + 1])
                    and (data_frame["RATE"][j + 1] > data_frame["RATE"][j + 2])
                    and (data_frame["RATE"][j + 2] > data_frame["RATE"][j + 3])
                ):
                    next_index = j + 4
                    break

            flare_range = np.array(data_frame["RATE"][lc_index + 4 : j])
            if len(flare_range) == 0:
                flare_range = np.array(data_frame["RATE"][lc_index + 3 : lc_index + 5])

            background_level = update_background(lc_index, data_frame, background_level)

            peak_value, peak_time = get_peak_info(flare_range, lc_index + 4, data_frame)

            flare_peak_values.append(peak_value)
            flare_peak_times.append(peak_time)
            flare_background_levels.append(background_level)

            flare_range_list = list(flare_range)
            index_max = flare_range_list.index(peak_value) + lc_index + 4

            try:
                curve = np.polyfit(
                    np.array(data_frame["RATE"][int(index_max - 4) : int(index_max)]),
                    np.array(data_frame["TIME"][int(index_max - 4) : int(index_max)]),
                    1,
                )
                poly = np.poly1d(curve)
                flare_start_points.append(poly(background_level))
                flare_end_times.append(
                    estimate_decay_end(
                        np.array(data_frame["RATE"][index_max : j + 6]),
                        np.array(data_frame["TIME"][index_max : j + 6]),
                        background_level,
                        data_frame["TIME"][index_max],
                    )
                )
            except:
                curve = np.polyfit(
                    np.array(data_frame["RATE"][lc_index:index_max]),
                    np.array(data_frame["TIME"][lc_index:index_max]),
                    1,
                )
                poly = np.poly1d(curve)
                flare_start_points.append(poly(background_level))
                flare_end_times.append(
                    estimate_decay_end(
                        np.array(data_frame["RATE"][index_max : j + 6]),
                        np.array(data_frame["TIME"][index_max : j + 6]),
                        background_level,
                        data_frame["TIME"][index_max],
                    )
                )

            flare_type.append(classify_flare(peak_value, background_level, flare_range))

            lc_index = next_index
        else:
            lc_index += 1

    return [
        flare_start_times,
        flare_type,
        flare_start_points,
        flare_peak_times,
        flare_end_times,
        flare_peak_values,
        flare_background_levels,
        flare_rates,
    ]


def get_peak_info(flare_range, offset, data_frame):
    peak_value = np.max(flare_range)
    peak_index = flare_range.tolist().index(peak_value) + offset
    return peak_value, data_frame["TIME"][peak_index]


def update_background(flare_index, data_frame, background):
    check_index = flare_index + 2
    while check_index > 0:
        if (
            data_frame["RATE"][flare_index] - data_frame["RATE"][check_index] > 900
            or check_index == 0
        ):
            next_index_average = np.mean(data_frame["RATE"][check_index:flare_index])
            if next_index_average / background > 1.5:
                return background
            elif (
                next_index_average / background < 1.5
                and np.max(data_frame["RATE"][flare_index + 4 :]) / next_index_average
                > 1.5
            ):
                return next_index_average
            elif data_frame["RATE"][flare_index] < next_index_average:
                return data_frame["RATE"][flare_index]
        check_index -= 1
    return background


def classify_flare(peak_value, background, flare_range):
    peak_diff = peak_value - background
    if 0 < peak_diff <= 1:
        return f"{(peak_diff * 10):.1f}A"
    elif peak_diff < 1:
        return f"{(peak_value - np.mean(flare_range)) * 10:.1f}A"
    elif 1 < peak_diff < 10:
        return f"{peak_diff:.1f}B"
    elif 10 <= peak_diff < 100:
        return f"{peak_diff / 10:.1f}C"
    elif 100 <= peak_diff < 1000:
        return f"{peak_diff / 100:.1f}M"
    else:
        return f"{peak_diff / 1000:.1f}X"


if __name__ == "__main__":
    data_file = input()
    detected_flares = load_file(data_file)
    flare_data = json.dumps(detected_flares, separators=(",", ":"))
    print(flare_data)
