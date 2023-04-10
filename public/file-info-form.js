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
const DeletedInput = MJBS.createInput(); // readonly

const MoveToBucketAlert = MJBS.createAlert();
const BucketSelect = cc("select", { classes: "form-select" });
const MoveToBucketBtn = MJBS.createButton("Move", "outline-primary");
const MoveToBucketGroup = cc("div", {
  classes: "input-group mb-3",
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
const SubmitBtn = MJBS.createButton("Submit");
const SubmitBtnAlert = MJBS.createAlert();

const EditFileForm = cc("form", {
  attr: { autocomplete: "off" },
  children: [
    MJBS.hiddenButtonElem(),

    m("div").addClass("text-center mt-0 mb-2").append(m(PicPreview).hide()),

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
    MJBS.createFormControl(DeletedInput, "Deleted", "檔案是否標記為刪除"),

    m(SubmitBtnAlert).addClass("my-3"),
    m("div")
      .addClass("text-center my-3")
      .append(
        m(SubmitBtn).on("click", (event) => {
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

function initEditFileForm(fileID, selfButton) {
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
      BucketInput.setVal(file.bucket_name);
      NameInput.setVal(file.name);
      NotesInput.setVal(file.notes);
      KeywordsInput.setVal(file.keywords);
      SizeInput.setVal(fileSizeToString(file.size));
      LikeInput.setVal(file.like);
      CTimeInput.setVal(file.ctime);
      UTimeInput.setVal(file.utime);
      CheckedInput.setVal(file.checked);
      DamagedInput.setVal(file.damaged);
      DeletedInput.setVal(file.deleted);

      MJBS.disable(IdInput);
      MJBS.disable(BucketInput);
      MJBS.disable(SizeInput);
      MJBS.disable(CheckedInput);
      MJBS.disable(DamagedInput);
      MJBS.disable(DeletedInput);

      EditFileForm.show();
      SubmitBtnAlert.clear();
      initBucketSelect(file.bucket_name);
    },
    onAlways: () => {
      FileInfoPageLoading.hide();
      if (selfButton) MJBS.enable(selfButton);
    },
  });
}

function BucketItem(bucket) {
  return cc("option", {
    id: "B-" + bucket.id,
    attr: { value: bucket.name, title: bucket.name },
    text: bucket.title,
  });
}

function getThumbnail(file_id) {
  axios
    .get(`/thumbs/${file_id}`)
    .then((resp) => {
      Thumbnail.show();
      Thumbnail.elem().attr({ src: "data:image/jpeg;base64," + resp.data });
      location = "#";
    })
    .catch((err) => {
      Thumbnail.hide();
      MJBS.focus(NotesInput);
      console.log(axiosErrToStr(err, errorData_toString));
    });
}

function getBuckets() {
  axiosGet({
    url: "/api/auto-get-buckets",
    alert: MoveToBucketAlert,
    onSuccess: (resp) => {
      FileInfoPageCfg.buckets = resp.data;
    },
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
      let val = bucket.name;
      if (bucket.name != bucket.title) val = `${bucket.name} (${bucket.title})`;
      BucketInput.setVal(val);
    } else {
      const item = BucketItem(bucket);
      BucketSelect.elem().append(m(item));
    }
  }
}

function updateFileItem(file) {
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
