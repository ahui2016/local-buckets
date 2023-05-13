const FileInfoPageCfg = {};
const FileInfoPageAlert = MJBS.createAlert();
const FileInfoPageLoading = MJBS.createLoading(null, "large");

const IdInput = MJBS.createInput("number", "required"); // readonly
const BucketInput = MJBS.createInput(); // readonly
const NameInput = MJBS.createInput("text", "required");
const NotesInput = MJBS.createInput();
const KeywordsInput = MJBS.createInput();
const SizeInput = MJBS.createInput(); // readonly
const LikeInput = MJBS.createInput("number");
const CTimeInput = MJBS.createInput("text", "required");
const UTimeInput = MJBS.createInput();
const CheckedInput = MJBS.createInput(); // readonly
const DamagedInput = MJBS.createInput(); // readonly
// const DeletedInput = MJBS.createInput(); // readonly

const MoveToBucketAlert = MJBS.createAlert();
const BucketSelect = cc("select", { classes: "form-select" });
const MoveToBucketBtn = MJBS.createButton("Move", "outline-primary");
const MoveToBucketGroup = cc("div", {
  classes: "input-group HideIfBackup mb-3",
  children: [
    span("Move to").addClass("input-group-text"),
    m(BucketSelect),
    m(MoveToBucketBtn).on("click", (event) => {
      event.preventDefault();
      const body = {
        file_id: IdInput.intVal(),
        bucket_name: BucketSelect.elem().val(),
      };
      if (!body.bucket_name) {
        MoveToBucketAlert.insert("warning", "請選擇一個倉庫");
        return;
      }

      MJBS.disable(MoveToBucketBtn);
      axiosPost({
        url: "/api/move-file-to-bucket",
        alert: MoveToBucketAlert,
        body: body,
        onSuccess: (resp) => {
          const file = resp.data;
          MoveToBucketAlert.clear().insert("success", "移動檔案成功!");
          initBucketSelect(file.bucket_name);
          updateFileItem(file);
        },
        onAlways: () => {
          MJBS.enable(MoveToBucketBtn);
        },
      });
    }),
  ],
});

const PicPreview = cc("img", {
  classes: "img-thumbnail",
  attr: { alt: "pic" },
});

const FileFormButtonsAlert = MJBS.createAlert();

const FileFormBadgesLeft = cc("div", {
  classes: "col-9 text-start",
  children: [span("❤").hide()],
});
const FileFormBadgesRight = cc("div", { classes: "col-9 text-end" });
const FileFormBadgesArea = cc("div", {
  classes: "mb-1",
  children: [m(FileFormBadgesLeft), m(FileFormBadgesRight)],
});

const FileFormButtonsArea = cc("div", {
  classes: "text-end",
  children: [
    MJBS.createLinkElem("#", { text: "DL" })
      .attr({ title: "download" })
      .addClass("ImageDownloadBtn btn btn-sm btn-light text-muted me-2"),
    MJBS.createLinkElem("#", { text: "small" })
      .attr({ title: "下載小圖" })
      .addClass("ImageDownloadSmallBtn btn btn-sm btn-light text-muted me-2"),
    MJBS.createLinkElem("#", { text: "view", blank: true })
      .attr({ title: "preview" })
      .addClass("ImagePreviewBtn btn btn-sm btn-light text-muted me-2"),
    MJBS.createLinkElem("#", { text: "del" })
      .attr({ title: "delete" })
      .addClass(
        "ImageDelBtn btn btn-sm btn-light text-muted HideIfBackup me-2"
      ),
    MJBS.createLinkElem("#", { text: "DELETE" })
      .addClass("ImageDangerDelBtn btn btn-sm btn-danger")
      .hide(),
  ],
});

const SubmitBtn = MJBS.createButton("Submit");
const SubmitBtnAlert = MJBS.createAlert();

const EditFileForm = cc("form", {
  attr: { autocomplete: "off" },
  children: [
    MJBS.hiddenButtonElem(),

    m("div").addClass("text-center mt-0 mb-2").append(m(PicPreview).hide()),

    m(FileFormBadgesArea).hide(),
    m(FileFormButtonsAlert).hide(),
    m(FileFormButtonsArea).hide(),

    MJBS.createFormControl(IdInput, "ID"),
    MJBS.createFormControl(
      BucketInput,
      "Bucket",
      "在下面選擇一個倉庫, 點擊 Move 按鈕, 可把檔案移至所選倉庫."
    ),
    m(MoveToBucketAlert).addClass("my-1"),
    m(MoveToBucketGroup),
    MJBS.createFormControl(NameInput, "File Name"),
    MJBS.createFormControl(NotesInput, "Notes", "關於該檔案的簡單描述"),
    MJBS.createFormControl(KeywordsInput, "Keywords", "關鍵詞, 用於輔助搜尋."),
    MJBS.createFormControl(SizeInput, "Size"),
    MJBS.createFormControl(
      LikeInput,
      "Like",
      "點讚數, 數字越大表示該檔案越重要."
    ),
    MJBS.createFormControl(
      CTimeInput,
      "CTime",
      "創建時間, 格式 2006-01-02 15:04:05+08:00"
    ),
    MJBS.createFormControl(UTimeInput, "UTime", "更新時間, 一般不需要修改."),
    MJBS.createFormControl(
      CheckedInput,
      "Checked",
      "上次檢查檔案完整性的時間."
    ),
    MJBS.createFormControl(DamagedInput, "Damaged", "檔案是否損壞"),
    // MJBS.createFormControl(DeletedInput, "Deleted", "檔案是否標記為刪除"),

    m(SubmitBtnAlert).addClass("my-3"),
    m("div")
      .addClass("text-center my-3")
      .append(
        m(SubmitBtn)
          .addClass("HideIfBackup")
          .on("click", (event) => {
            event.preventDefault();

            const body = {
              id: IdInput.intVal(),
              name: NameInput.val(),
              notes: NotesInput.val(),
              keywords: KeywordsInput.val(),
              like: LikeInput.intVal(),
              ctime: CTimeInput.val(),
              utime: UTimeInput.val(),
            };

            MJBS.disable(SubmitBtn); // --------------------------- disable
            axiosPost({
              url: "/api/update-file-info",
              alert: SubmitBtnAlert,
              body: body,
              onSuccess: (resp) => {
                const file = resp.data;
                SubmitBtnAlert.clear().insert("success", "修改成功");
                updateFileItem(file);
              },
              onAlways: () => {
                MJBS.enable(SubmitBtn); // ------------------------ enable
              },
            });
          })
      ),
  ],
});

function initFileFormButtons(fileID) {
  const downladBtnID = ".ImageDownloadBtn";
  const smallBtnID = ".ImageDownloadSmallBtn";
  const previewBtnID = ".ImagePreviewBtn";
  const delBtnID = ".ImageDelBtn";
  const dangerDelBtnID = `.ImageDangerDelBtn`;

  FileFormButtonsAlert.clear();
  $(delBtnID).show();
  $(dangerDelBtnID).hide();

  $(downladBtnID)
    .off()
    .on("click", (event) => {
      event.preventDefault();
      MJBS.disable(downladBtnID);
      event.currentTarget.style.pointerEvents = "none";
      axiosPost({
        url: "/api/download-file",
        alert: FileFormButtonsAlert,
        body: { id: fileID },
        onSuccess: () => {
          FileFormButtonsAlert.insert(
            "success",
            `成功下載到 waiting 資料夾 ${PageConfig.waitingFolder}`
          );
        },
        onAlways: () => {
          MJBS.enable(downladBtnID);
        },
      });
    });

  $(smallBtnID)
    .off()
    .on("click", (event) => {
      event.preventDefault();
      MJBS.disable(smallBtnID);
      event.currentTarget.style.pointerEvents = "none";
      axiosPost({
        url: "/api/download-small-pic",
        alert: FileFormButtonsAlert,
        body: { id: fileID },
        onSuccess: () => {
          FileFormButtonsAlert.insert(
            "success",
            `成功下載到 waiting 資料夾 ${PageConfig.waitingFolder}`
          );
        },
        onAlways: () => {
          MJBS.enable(smallBtnID);
        },
      });
    });

  if (PageConfig.showSmallBtn) {
    $(smallBtnID).show();
  } else {
    $(smallBtnID).hide();
  }

  $(previewBtnID).attr({ href: "/file/" + fileID });

  $(delBtnID)
    .off()
    .on("click", (event) => {
      event.preventDefault();
      MJBS.disable(delBtnID);
      FileFormButtonsAlert.clear().insert(
        "warning",
        "等待 3 秒, 點擊紅色的 DELETE 按鈕刪除檔案 (注意, 一旦刪除, 不可恢復!)."
      );
      setTimeout(() => {
        MJBS.enable(delBtnID);
        $(delBtnID).hide();
        $(dangerDelBtnID).show();
      }, 2000);
    });

  $(dangerDelBtnID)
    .off()
    .on("click", (event) => {
      event.preventDefault();
      MJBS.disable(FileFormButtonsArea);
      axiosPost({
        url: "/api/delete-file",
        alert: FileFormButtonsAlert,
        body: { id: fileID },
        onSuccess: () => {
          $("#F-" + fileID).hide();
          EditFileForm.hide();
          FileInfoPageAlert.clear().insert("success", "該檔案已被刪除");
        },
        onAlways: () => {
          MJBS.enable(FileFormButtonsArea);
        },
      });
    });
}

function initEditFileForm(fileID, selfButton, onlyImages) {
  EditFileForm.hide();
  FileInfoPageLoading.show();
  FileInfoPageAlert.clear();
  SubmitBtnAlert.clear();
  MoveToBucketAlert.clear();

  if (onlyImages) {
    FileFormBadgesArea.show();
    FileFormButtonsAlert.show();
    FileFormButtonsArea.show();
    initFileFormButtons(fileID);
  }
  if (selfButton) MJBS.disable(selfButton);
  axiosPost({
    url: "/api/file-info",
    alert: FileInfoPageAlert,
    body: { id: fileID },
    onSuccess: (resp) => {
      const file = resp.data;

      if (file.type.startsWith("image")) {
        PicPreview.show();
        PicPreview.elem().attr({ src: `/file/${file.id}` });
      } else {
        PicPreview.hide();
        MJBS.focus(NotesInput);
      }

      IdInput.setVal(file.id);
      // BucketInput.setVal(bucket_name); // 在 initBucketSelect 中賦值
      NameInput.setVal(file.name);
      NotesInput.setVal(file.notes);
      KeywordsInput.setVal(file.keywords);
      SizeInput.setVal(fileSizeToString(file.size));
      LikeInput.setVal(file.like);
      CTimeInput.setVal(file.ctime);
      UTimeInput.setVal(file.utime);
      CheckedInput.setVal(file.checked);
      DamagedInput.setVal(file.damaged);
      // DeletedInput.setVal(file.deleted);

      MJBS.disable(IdInput);
      MJBS.disable(BucketInput);
      MJBS.disable(SizeInput);
      MJBS.disable(CheckedInput);
      MJBS.disable(DamagedInput);
      // MJBS.disable(DeletedInput);

      EditFileForm.show();
      initBucketSelect(file.bucket_name);
    },
    onAlways: () => {
      if (PageConfig.projectInfo.is_backup) {
        $(".HideIfBackup").hide();
      }
      FileInfoPageLoading.hide();
      if (selfButton) MJBS.enable(selfButton);
      window.location = PicPreview.id;
    },
  });
}

function BucketItem(bucket) {
  let text = bucket.title;
  if (bucket.encrypted) text = "🔒" + text;
  return cc("option", {
    id: "B-" + bucket.id,
    attr: { value: bucket.name, title: bucket.name },
    text: text,
  });
}

const CurrentBucketAlert = cc("div", {
  classes: "alert alert-info",
  attr: { role: "alert" },
});

function getBuckets(alert) {
  return new Promise((resolve) => {
    axiosGet({
      url: "/api/auto-get-buckets",
      alert: alert,
      onSuccess: (resp) => {
        const buckets = resp.data;
        const currentID = getUrlParam("bucket");
        let currentName = getUrlParam("bucketname");
        if (!currentName) currentName = "";

        if (!currentID && !currentName) {
          resolve(buckets);
          return;
        }

        let hasBucket = false;
        for (const bucket of buckets) {
          if (
            currentID == bucket.id ||
            currentName.toLowerCase() == bucket.name.toLowerCase()
          ) {
            hasBucket = true;
            const name =
              bucket.name != bucket.title
                ? `bucket.name(${bucket.title})`
                : bucket.name;
            CurrentBucketAlert.show();
            CurrentBucketAlert.elem().text(`正在瀏覽倉庫: ${name}`);
          }
        }
        if (!hasBucket) {
          alert.insert("danger", `找不到倉庫: ${currentID} ${currentName}`);
        }
        resolve(buckets);
      },
    });
  });
}

function initBucketSelect(currentbucketName) {
  BucketSelect.elem().html("");
  BucketSelect.elem().append(
    m("option")
      .prop("selected", true)
      .attr({ value: "" })
      .text("點擊此處選擇倉庫...")
  );

  for (const bucket of FileInfoPageCfg.buckets) {
    if (bucket.name == currentbucketName) {
      let bucketVal = bucket.name;
      if (bucket.encrypted) bucketVal = "🔒" + bucketVal;
      if (bucket.name != bucket.title)
        bucketVal = `${bucketVal} (${bucket.title})`;
      BucketInput.setVal(bucketVal);
    } else {
      const item = BucketItem(bucket);
      BucketSelect.elem().append(m(item));
    }
  }
}

function updateFileItem(file) {
  if (PageConfig.picsInBucket) {
    $("#F-" + file.id).hide();
    return;
  }
  const item = FileItem(file);
  item.elem().replaceWith(m(item));
  item.init();
}

const rootMarginLeft = "550px";

const FileEditCanvas = cc("div", {
  classes: "offcanvas offcanvas-start",
  css: { width: rootMarginLeft },
  attr: {
    "data-bs-scroll": true,
    "data-bs-backdrop": false,
    tabindex: -1,
  },
  children: [
    m("div")
      .addClass("offcanvas-header")
      .append(
        m("h5").addClass("offcanvas-title").text("File Info (檔案屬性)"),
        m("button").addClass("btn-close").attr({
          type: "button",
          "data-bs-dismiss": "offcanvas",
          "aria-label": "Close",
        })
      ),
    m("div")
      .addClass("offcanvas-body")
      .append(
        m(FileInfoPageAlert),
        m(FileInfoPageLoading).addClass("my-5"),
        m(EditFileForm).hide()
      ),
  ],
});

function getWaitingFolder() {
  axiosGet({
    url: "/api/waiting-folder",
    alert: PageAlert,
    onSuccess: (resp) => {
      PageConfig.waitingFolder = resp.data.text;
    },
  });
}

function setExport(b) {
  axiosPost({
    url: "/api/set-export",
    body: { text: "" + b },
    alert: PageAlert,
    onSuccess: (resp) => {
      if (resp.data) {
        console.log("OK, 已設定為: 下載同時導出.");
      } else {
        console.log("OK, 已設定為: 只下載, 不導出.");
      }
    },
  });
}
