import http from 'k6/http';
import { check, sleep } from 'k6';
import { FormData } from 'https://jslib.k6.io/formdata/0.0.2/index.js';

export let options = {
    vus: 1000,
    duration: '5m',
};

const csvfile = open('file.csv', 'b');
export default function () {
    const formdata = new FormData();
    formdata.append('file', http.file(csvfile, 'file.csv', 'text/csv'));

    const uploadResponse = http.post('http://localhost:8000/upload', formdata.body() , {
        headers: { 'Content-Type': 'multipart/form-data; boundary=' + formdata.boundary },
    });

    check(uploadResponse, {
        'status was 200': (r) => r.status === 200,
    });
    sleep(1);
}