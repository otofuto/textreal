function post(url, data) {
	return new Promise((resolve, reject) => {
		fetch(url, {
			method: 'post',
			body: data == null ? new FormData() : data,
			credentials: "include"
		}).then(res => {
			if (res.status == 200)
				return res.text();
			return 'null';
		}).then(text => {
			try {
				let result = JSON.parse(text);
				if (result != null) {
					resolve(result);
				} else {
					reject(null);
				}
			} catch (err) {
				console.error(err);
				console.log(text);
				reject(null);
			}
		}).catch(err => {
			console.error(err);
			reject(err);
		});
	});
}

function get(url) {
	return new Promise((resolve, reject) => {
		fetch(url)
		.then(res => {
			if (res.status == 200)
				return res.json();
			return null;
		}).then(result => {
			if (result != null) {
				resolve(result);
			} else {
				reject(null);
			}
		}).catch(err => {
			console.error(err);
			reject(err);
		});
	});
}

function put(url, data) {
	return new Promise((resolve, reject) => {
		fetch(url, {
			method: 'put',
			body: data == null ? new FormData() : data,
			credentials: "include"
		}).then(res => {
			if (res.status == 200)
				return res.json();
			return null;
		}).then(result => {
			if (result != null) {
				resolve(result);
			} else {
				reject(null);
			}
		}).catch(err => {
			console.error(err);
			reject(err);
		});
	});
}

function del(url, data) {
	return new Promise((resolve, reject) => {
		fetch(url, {
			method: 'delete',
			body: data == null ? new FormData() : data,
			credentials: "include"
		}).then(res => {
			if (res.status == 200)
				return res.json();
			return null;
		}).then(result => {
			if (result != null) {
				resolve(result);
			} else {
				reject(null);
			}
		}).catch(err => {
			console.error(err);
			reject(err);
		});
	});
}

function formDisabled(form, dis) {
	if (dis) {
		Array.from(form.getElementsByTagName('input')).forEach(elm => elm.setAttribute('disabled', ''));
		Array.from(form.getElementsByTagName('textarea')).forEach(elm => elm.setAttribute('disabled', ''));
		Array.from(form.getElementsByTagName('button')).forEach(elm => elm.setAttribute('disabled', ''));
		Array.from(form.getElementsByTagName('select')).forEach(elm => elm.setAttribute('disabled', ''));
        Array.from(form.querySelectorAll('input[type="checkbox"]')).forEach(elm => elm.setAttribute('onclick', 'return false;'));
        Array.from(form.querySelectorAll('input[type="radiobutton"]')).forEach(elm => elm.setAttribute('onclick', 'return false;'));
	} else {
		Array.from(form.getElementsByTagName('input')).forEach(elm => elm.removeAttribute('disabled'));
		Array.from(form.getElementsByTagName('textarea')).forEach(elm => elm.removeAttribute('disabled'));
		Array.from(form.getElementsByTagName('button')).forEach(elm => elm.removeAttribute('disabled'));
		Array.from(form.getElementsByTagName('select')).forEach(elm => elm.removeAttribute('disabled'));
        Array.from(form.querySelectorAll('input[type="checkbox"]')).forEach(elm => elm.removeAttribute('onclick'));
        Array.from(form.querySelectorAll('input[type="radiobutton"]')).forEach(elm => elm.removeAttribute('onclick'));
	}
}

function formatdate(str, timeView = true) {
	let dt = new Date(str);
	let ret = dt.getFullYear() + '年 ' +
		(dt.getMonth() + 1) + '月 ' + dt.getDate() + '日';
	if (timeView) ret += ' ' + frontZero(dt.getHours()) + ':' + frontZero(dt.getMinutes());
	return ret;
}

function object2form(obj, form) {
	for (let i = 0; i < Object.keys(obj).length; i++) {
		let k = Object.keys(obj)[i];
		let v = obj[k];
		if (typeof v.Valid == 'boolean') {
			if (typeof v.String != 'undefined')
				v = v.String;
			else if (typeof v.Int64 != 'undefined')
				v = v.Int64;
		}
		if (k.endsWith('[]') && Array.isArray(v)) {
			v.forEach(v2 => {
				form.querySelectorAll('[name="' + k + '"]').forEach(input => {
					if (!input.checked && input.value == v2) input.click();
				});
			});
		} else if (Array.isArray(v)) {
			v.forEach(v2 => {
				form.querySelectorAll('[name="' + k + '[]"]').forEach(input => {
					if (!input.checked && input.value == v2) input.click();
				});
			});
		} else {
			form.querySelectorAll('[name="' + k + '"]').forEach(input => {
				switch (input.getAttribute('type')) {
					case 'checkbox':
						if (!input.checked) input.click();
						break;
					case 'radio':
						if (input.value == v) input.click();
						break;
					case 'file':
						break;
					case 'datetime-local':
						input.value = v.replace(' ', 'T');
						break;
					default:
						input.value = v;
						break;
				}
			});
		}
	}
}

function frontZero(s) {
	if ((s - 0) < 10) s = '0' + s;
	return s;
}