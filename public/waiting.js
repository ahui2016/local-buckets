const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Local-Buckets" }),
        span(" .. Waiting (等待上傳)")
      ),
    m("div")
      .addClass("col text-end")
      .append(
        MJBS.createLinkElem("#", { text: "Link1" }).addClass("Link1"),
        " | ",
        MJBS.createLinkElem("#", { text: "Link2" }).addClass("Link2")
      )
  );

const PageAlert = MJBS.createAlert();
const PageLoading = MJBS.createLoading(null, "large");

const BucketSelect = cc("select", {
  classes: "form-select",
  children: [
    m("option")
      .prop("selected", true)
      .attr({ value: "" })
      .text("請選擇一個倉庫..."),
  ],
});

const BucketSelectGroup = cc("div", {
  classes: "input-group input-group-lg",
  children: [span("Bucket").addClass("input-group-text"), m(BucketSelect)],
});

function BucketItem(bucket) {
  return cc("option", {
    id: "B-" + bucket.id,
    attr: { value: bucket.id, title: bucket.id },
    text: bucket.title,
  });
}

const WaitingFileList = cc("ul", { classes: "list-group list-group-flush" });

function FileItem(file) {
  return cc("li", {
    id: "F-" + file.checksum,
    classes: "list-group-item",
    children: [
      span(file.size)
        .addClass("text-muted me-2")
        .text(fileSizeToString(file.size)),
      span(file.name),
    ],
  });
}

const UploadButton = MJBS.createButton("Upload");
const UploadAlert = MJBS.createAlert();
const UploadButtonArea = cc("div", {
  classes: "text-center",
  children: [
    m(UploadAlert),
    m(UploadButton).on("click", (event) => {
      event.preventDefault();
      const bucketid = BucketSelect.elem().val();
      if (!bucketid) {
        UploadAlert.insert("warning", "請選擇一個倉庫");
        return;
      }
      MJBS.disable(UploadButton); // --------------------- disable
      axiosPost({
        url: "/api/upload-new-files",
        body: { bucketid: bucketid },
        alert: UploadAlert,
        onSuccess: () => {
          UploadAlert.clear().insert("success", "上傳成功");
        },
        onAlways: () => {
          MJBS.enable(UploadButton); // ------------------- enable
        },
      });
    }),
  ],
});

$("#root")
  .css(RootCss)
  .append(
    navBar.addClass("my-3"),
    m(PageAlert).addClass("my-5"),
    m(SameNameRadioCard).addClass("my-5").hide(),
    m(BucketSelectGroup).addClass("my-5").hide(),
    m(WaitingFileList).addClass("my-5"),
    m(UploadButtonArea).addClass("my-5").hide(),
    m(PageLoading).addClass("my-5")
  );

init();

async function init() {
  if ((await initBucketSelect()) == "fail") {
    return;
  }
  getWaitingFolder();
  getWaitingFiles();
}

function initBucketSelect() {
  return new Promise((resolve) => {
    axiosGet({
      url: "/api/all-buckets",
      alert: PageAlert,
      onSuccess: (resp) => {
        const buckets = resp.data;
        if (buckets && buckets.length > 0) {
          MJBS.appendToList(BucketSelect, buckets.map(BucketItem));
          resolve("success");
        } else {
          PageAlert.insert(
            "warning",
            "沒有倉庫, 請返回首頁, 點擊 Create Bucket 新建倉庫."
          );
          resolve("fail");
        }
      },
    });
  });
}

function getWaitingFiles() {
  axios
    .get("/api/waiting-files")
    .then((resp) => {
      const files = resp.data;
      if (files && files.length > 0) {
        BucketSelectGroup.show();
        UploadButtonArea.show();
        MJBS.appendToList(WaitingFileList, files.map(FileItem));
        PageAlert.insert(
          "light",
          "這裡列出的檔案清單僅供參考, 實際上傳檔案以 waiting 資料夾為準.",
          "no-time"
        );
      } else {
        PageAlert.insert("info", "沒有等待上傳的檔案");
        PageAlert.insert(
          "info",
          "請把檔案放到 waiting 資料夾, 然後刷新本頁面."
        );
      }
    })
    .catch((err) => {
      getWaitingFilesErrorHandler(err, PageAlert);
    })
    .then(() => {
      PageLoading.hide();
    });
}

function getWaitingFolder() {
  axiosGet({
    url: "/api/waiting-folder",
    alert: PageAlert,
    onSuccess: (resp) => {
      PageAlert.insert(
        "light",
        `waiting 資料夾 ➡️ ${resp.data.text}`,
        "no-time"
      );
    },
  });
}

function getWaitingFilesErrorHandler(err, alert) {
  if (err.response) {
    const respData = err.response.data;
    if (typeof respData === "string") {
      alert.insert("danger", respData);
      return;
    }
    if (err.response.data.errType == "ErrSameNameFiles") {
      const errSameName = respData;
      console.log(errSameName);
      alert.insert("warning", "檔案名稱重複, 請處理.");
      SameNameRadioCard.show();
      SameNameRadioCard.init(errSameName.file);
      return;
    }
    alert.insert("danger", JSON.stringify(respData));
    return;
  }

  if (err.request) {
    const errMsg =
      err.request.status +
      " The request was made but no response was received.";
    alert.insert("danger", errMsg);
    return;
  }

  alert.insert("danger", err.message);
}
