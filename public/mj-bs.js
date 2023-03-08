// 这些 class 只是为了方便生成文档，不实际使用。
class LinkOptions {}

const MJBS = {};

/**
 * obj 是 mjComponent 或 mjComponent 的 id.
 * @param {mjComponent | string} obj
 */
MJBS.disable = function (obj) {
  const id = typeof obj == "string" ? obj : obj.id;
  const nodeName = $(id).prop("nodeName");
  if (nodeName == "BUTTON" || nodeName == "INPUT") {
    $(id).prop("disabled", true);
  } else {
    $(id).css("pointer-events", "none");
  }
};

/**
 * obj 是 mjComponent 或 mjComponent 的 id.
 * @param {mjComponent | string} obj
 */
MJBS.enable = function (obj) {
  const id = typeof obj == "string" ? obj : obj.id;
  const nodeName = $(id).prop("nodeName");
  if (nodeName == "BUTTON" || nodeName == "INPUT") {
    $(id).prop("disabled", false);
  } else {
    $(id).css("pointer-events", "auto");
  }
};

/**
 * @param {mjComponent} list
 * @param {mjComponent[]} items
 */
MJBS.prependToList = function (list, items) {
  items.forEach((item) => {
    list.elem().prepend(m(item));
    if (item.init) item.init();
  });
};

/**
 * @param {mjComponent} list
 * @param {mjComponent[]} items
 */
MJBS.appendToList = function (list, items) {
  items.forEach((item) => {
    list.elem().append(m(item));
    if (item.init) item.init();
  });
};

/**
 * @param {mjElement | mjComponent} obj
 * @param {"trim"?} trim
 * @returns {string}
 */
MJBS.valOf = function (obj, trim) {
  let s = "elem" in obj ? obj.elem().val() : obj.val();
  return trim == "trim" ? s.trim() : s;
};

/**
 * @param {mjElement | mjComponent} obj
 * @param {number?} timeout default=300
 */
MJBS.focus = function (obj, timeout = 300) {
  if ("elem" in obj) obj = obj.elem();
  setTimeout(() => {
    obj.trigger("focus");
    obj[0].setSelectionRange(1000, 1000);
  }, timeout);
};

/**
 * msgType: primary/secondary/success/danger/warning/info/light/dark
 */
MJBS.createAlert = function () {
  const alert = cc("div");

  alert.insertElem = (elem) => {
    alert.elem().prepend(elem);
  };

  /**
   * msgType: primary/secondary/success/danger/warning/info/light/dark
   * @param {string} msgType
   * @param {string} msg
   * @param {boolean} prefix_time
   */
  alert.insert = (msgType, msg, prefix_time=true) => {
    if (prefix_time) msg = `${dayjs().format("HH:mm:ss")} ${msg}`;
    if (msgType == "danger") console.log(msg);

    const dismissBtn = m("button").addClass("btn-close").attr({
      type: "button",
      "data-bs-dismiss": "alert",
      "aria-label": "Close",
    });

    const elem = m("div")
      .addClass(`alert alert-${msgType} alert-dismissible fade show my-1`)
      .attr({ role: "alert" })
      .append(span(msg), dismissBtn);

    alert.insertElem(elem);
  };

  alert.clear = () => {
    alert.elem().html("");
  };

  return alert;
};

/**
 * 使用方法:
 * ```
 * const Toasts = MJBS.createToasts();
 * $('#root').append(m(IDToasts)); // 必須立即實體化.
 * const Toast = Toasts.new();
 * // Toast.setTitle(title);
 * Toast.popup(body, title?, color?);
 * ```
 */
MJBS.createToasts = function (classes) {
  if (!classes) classes = "toast-container position-fixed";
  const self = cc("div", { classes: classes });

  self.new = () => {
    const Toast = createToast();
    self.elem().prepend(m(Toast));
    return Toast;
  };

  return self;
};

function createToast() {
  const self = cc("div", {
    classes: "toast text-bg-primary",
    attr: { role: "alert", "aria-live": "assertive", "aria-atomic": "true" },
    children: [
      m("div")
        .addClass("toast-header")
        .append(
          // m("img").addClass("rounded me-2").attr({ src: "...", alt: "..." }),
          m("strong").addClass("me-auto"),
          m("small"),
          m("button").addClass("btn-close").attr({
            type: "button",
            "data-bs-dismiss": "toast",
            "aria-label": "Close",
          })
        ),
      m("div").addClass("toast-body"),
    ],
  });

  self.setTitle = (title) => {
    self.find("strong").text(title);
  };

  /**
   * @param {mjElement | string} body
   * @param {string?} title
   * @param {string?} color default: 'primary'
   */
  self.popup = (body, title, color) => {
    if (!color) color = "primary";
    self.elem().removeClass().addClass(`toast text-bg-${color}`);
    if (title) self.find("strong").text(title);
    self.find("small").text(dayjs().format("HH:mm:ss"));
    if (typeof body == "string") body = span(body);
    self.find(".toast-body").html("").append(body);
    const toast = new bootstrap.Toast(self.elem()[0]);
    toast.show();
  };

  return self;
}

/**
 * 使用方法:
 * ```
 * const MyModal = MJBS.createStaticModal(title, body, footer);
 * const modalBtn = cc('button', {
 *   text: 'Launch the modal',
 *   classes: 'btn btn-primary',
 *   attr: {type:'button', 'data-bs-toggle':'modal', data-bs-target: MyModal.id}
 * });
 * ```
 * @param {string} title
 * @param {mjElement | string} body
 * @param {mjElement | string | null} footer
 */
MJBS.createStaticModal = function (title, body, footer) {
  if (typeof body == "string") {
    body = m("p").text(body);
  }

  const modalContent = m("div")
    .addClass("modal-content")
    .append(
      m("div")
        .addClass("modal-header")
        .append(
          m("h5").addClass("modal-title").text(title),
          m("button").addClass("btn-close").attr({
            type: "button",
            "data-bs-dismiss": "modal",
            "aria-label": "Close",
          })
        ),
      m("div").addClass("modal-body").append(body)
    );

  if (footer) {
    if (typeof footer == "string") {
      footer = m("p").text(footer);
    }
    modalContent.append(m("div").addClass("modal-footer").append(footer));
  }

  const modalDialog = m("div").addClass("modal-dialog").append(modalContent);

  const modal = cc("div", {
    classes: "modal",
    children: [modalDialog],
  });

  return modal;
};

/**
 * 使用方法:
 * ```
 * const MyModal = MJBS.createModal('lg');
 * MyModal.popup(title, body, footer);
 * ```
 * @param {string?} size 'xl' | 'lg' | 'sm'
 * @returns {mjComponent}
 */
MJBS.createModal = function (size) {
  let classes = "modal-dialog";
  if (size) {
    classes = `modal-dialog modal-${size}`;
  }
  const modal = cc("div", {
    classes: "modal",
    children: [
      m("div")
        .addClass(classes)
        .append(
          m("div")
            .addClass("modal-content")
            .append(
              m("div")
                .addClass("modal-header")
                .append(
                  m("h5").addClass("modal-title"),
                  m("button").addClass("btn-close").attr({
                    type: "button",
                    "data-bs-dismiss": "modal",
                    "aria-label": "Close",
                  })
                ),
              m("div").addClass("modal-body")
            )
        ),
    ],
  });

  /**
   * @param {string} title
   * @param {mjElement | string} body
   * @param {mjElement | string | null} footer
   * @param {object?} options
   */
  modal.popup = (title, body, footer, options) => {
    modal.find(".modal-title").text(title);

    if (typeof body == "string") {
      body = m("p").text(body);
    }

    modal.find(".modal-body").html("").append(body);
    modal.find(".modal-footer").remove();

    if (footer) {
      if (typeof footer == "string") {
        footer = m("p").text(footer);
      }
      modal
        .find(".modal-content")
        .append(m("div").addClass("modal-footer").append(footer));
    }

    const modalElem = new bootstrap.Modal(modal.id, options);
    modalElem.show();
  };

  return modal;
};

/**
 * @param {string} href
 * @param {LinkOptions?} options `{text?: string, title?: string, blank?: boolean}`
 * @returns {mjElement}
 */
MJBS.createLinkElem = function (href, options) {
  if (!options) {
    return m("a").text(href).attr("href", href);
  }
  if (!options.text) options.text = href;
  const link = m("a").text(options.text).attr("href", href);
  if (options.title) link.attr("title", options.title);
  if (options.blank) link.attr("target", "_blank");
  return link;
};

/**
 * @param {string} type default = "text"
 * @param {'required'?} required
 * @param {string?} id
 * @returns {mjComponent}
 */
MJBS.createInput = function (type = "text", required = null, id = null) {
  return cc("input", {
    id: id,
    classes: "form-control",
    attr: { type: type },
    prop: { required: required == "required" ? true : false },
  });
};

/**
 * @param {number} rows default = 3
 * @returns {mjComponent}
 */
MJBS.createTextarea = function (rows = 3, id = null) {
  return cc("textarea", {
    id: id,
    classes: "form-control",
    attr: { rows: rows },
  });
};

/**
 * 主要用来给 input 或 textarea 包裹一层。
 * @param {mjComponent} comp 用 createInput 或 createTextarea 函数生成的组件。
 * @param {string} labelText
 * @param {string | mjElement | null} description
 * @param {string} classes default = "mb-3"
 * @returns {mjElement}
 */
MJBS.createFormControl = function (
  comp,
  labelText,
  description,
  classes = "mb-3"
) {
  const formControl = m("div")
    .addClass(classes)
    .append(
      m("label")
        .addClass("form-label")
        .attr({ for: comp.raw_id })
        .text(labelText),
      m(comp)
    );

  if (!description) return formControl;

  let descElem = description;

  if (typeof description == "string") {
    descElem = m("div").addClass("form-text").text(description);
  }
  formControl.append(descElem);
  return formControl;
};

/**
 * 这个按钮是隐藏不用的，为了防止按回车键提交表单
 */
MJBS.hiddenButtonElem = function () {
  return m("button")
    .text("submit")
    .attr({ type: "submit" })
    .hide()
    .on("click", (e) => {
      e.preventDefault();
      return false;
    });
}

/**
 * color: primary, secondary, success, danger, warning, info, light, dark, link
 * @param {string} name
 * @param {string} color
 * @param {string} type
 * @returns {mjComponent}
 */
MJBS.createButton = function (name, color = 'primary', type = 'button') {
  return cc("button", {
    text: name,
    classes: `btn btn-${color}`,
    attr: { type: type },
  });
}

/**
 * 使用方法:
 * ```
 * const PageNav = MJBS.createSimplePageNav('lg', 'center');
 * PageNav.setThisPage(text);
 * PageNav.setPreviousPage(href, text);
 * PageNav.setNextPage(href, text);
 * ```
 * @param {string?} size 'lg' | 'sm' | null
 * @param {string?} align 'center' | 'end' | null
 * @returns {mjComponent}
 */
MJBS.createSimplePageNav = (size, align) => {
  const PreviousPage = cc("a", { classes: "page-link", attr: { href: "#" } });
  const ThisPage = cc("a", { classes: "page-link disabled text-bg-light text-muted" });
  const NextPage = cc("a", { classes: "page-link", attr: { href: "#" } });
  
  let classes = "pagination";
  if (size) classes = `${classes} pagination-${size}`;
  if (align) classes = `${classes} justify-content-${align}`;

  const PageNav = cc("nav", {
    attr: { "aria-label": "Page navigation" },
    children: [
      m("ul")
        .addClass(classes)
        .append(
          m("li").addClass("page-item").append(m(PreviousPage)),
          m("li").addClass("page-item").append(m(ThisPage)),
          m("li").addClass("page-item").append(m(NextPage))
        ),
    ],
  });
  
  PageNav.setThisPage = (text) => {
    ThisPage.elem().text(text);
  };
  PageNav.setPreviousPage = (href, text) => {
    PreviousPage.elem().attr({href: href}).text(text);
  };
  PageNav.setNextPage = (href, text) => {
    NextPage.elem().attr({href: href}).text(text);
  };
  return PageNav;
};

/**
 * 如果 id 以数字开头，就需要使用 elemID 给它改成以字母开头，
 * 因为 DOM 的 ID 不允许以数字开头。
 * @param {string} id
 * @param {string} prefix
 * @returns {string}
 */
MJBS.elemID = function (id, prefix = "e") {
  return `${prefix}${id}`;
}

// 以下是一些 helper functions

/**
 * @param {string} text
 * @param {mjComponent?} alert
 * @param {string?} successMsg
 */
function copyToClipboard(text, alert, successMsg) {
  if (!successMsg) successMsg = "複製成功";
  navigator.clipboard.writeText(text).then(
    () => {
      if (alert) alert.insert("success", successMsg);
    },
    () => {
      if (alert) alert.insert("danger", "複製失敗");
    }
  );
}

function copyToClipboard2(text, onSuccess, onFail) {
  navigator.clipboard.writeText(text).then(onSuccess, onFail);
}

// https://axios-http.com/docs/handling_errors
function axiosErrToStr(err, data2str) {
  if (err.response) {
    if (err.response.status == 500) {
      return "500 Internal Server Error";
    }
    const dataText = data2str(err.response.data);
    // err.response.data.detail 裏的 detail 與後端對應.
    return `[${err.response.status}] ${dataText}`;
  }
  if (err.request) {
    return (
      err.request.status + " The request was made but no response was received."
    );
  }
  return err.message;
}

function errorData_toString(data) {
  return JSON.stringify(data);
}

// api = HTTP-Get("/openapi.json")
// validationError = api.components.schemas.ValidationError
function validationErrorData_toString(data) {
  if (typeof data.detail === "string") {
    return data.detail;
  }
  const detail = data.detail[0];
  const loc = JSON.stringify(detail.loc);
  return `錯誤位置: ${loc}; 錯誤原因: ${detail.msg}`;
}

// 一般設為 errorData_toString
const defaultData2str = validationErrorData_toString;

/**
 * axios get with default error handler.
 * axiosGetOptions {url, alert, data2str, onSuccess, onAlways}
 */
function axiosGet(options) {
  if (!options.data2str) options.data2str = defaultData2str;
  axios
    .get(options.url)
    .then(options.onSuccess)
    .catch((err) => {
      options.alert.insert("danger", axiosErrToStr(err, options.data2str));
    })
    .then(() => {
      if (options.onAlways) onAlways();
    });
}

/**
 * axios post with default error handler.
 * axiosGetOptions {url, body, alert, data2str, onSuccess, onAlways}
 */
function axiosPost(options) {
  if (!options.data2str) options.data2str = defaultData2str;
  axios
    .post(options.url, options.body)
    .then(options.onSuccess)
    .catch((err) => {
      options.alert.insert("danger", axiosErrToStr(err, options.data2str));
    })
    .then(() => {
      if (options.onAlways) onAlways();
    });
}

/**
 * @param {string} s
 * @returns {boolean}
 */
function hasWhiteSpace(s) {
  return /\s/g.test(s);
}

/**
 * 获取地址栏的参数。
 * @param {string} name
 * @returns {string | null}
 */
function getUrlParam(name) {
  const queryString = new URLSearchParams(document.location.search);
  return queryString.get(name);
}

// 以下與 mj-bs.js 無關.

const DateFormatYMD = "YYYY-MM-DD";

// 糊塗記帳預設金額
const predefinedAmounts = [
  0, 5, 10, 15, 20, 30, 40, 50, 60, 70, 80, 90, 100, 150, 200, 300, 400, 500,
  600, 700, 800, 900, 1000, 1500, 2000, 3000, 4000, 5000, 6000, 7000, 8000,
  9000, 10000,
];

function showPageNav_by_day(today, pageNav) {
  const oneDay = 24 * 60 * 60;
  const timestamp = dayjs(today).unix();
  const tomorrow = dayjs.unix(timestamp + oneDay).format(DateFormatYMD);
  const yesterday = dayjs.unix(timestamp - oneDay).format(DateFormatYMD);

  pageNav.show();
  pageNav.setThisPage(today);
  pageNav.setPreviousPage(`?day=${yesterday}`, `⬅️${yesterday}`);
  pageNav.setNextPage(`?day=${tomorrow}`, `${tomorrow}➡️`);
}
