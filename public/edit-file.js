$("title").text("Edit File Attributes (修改檔案屬性) - Local Buckets");

const fileID = getUrlParam("id");

const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Local-Buckets" }),
        span(" .. Edit File Attributes (修改檔案屬性)")
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

const IdInput = MJBS.createInput("number", "required"); // readonly
const BucketInput = MJBS.createInput(); // readonly
const NameInput = MJBS.createInput("text", "required");
const NotesInput = MJBS.createInput();
const KeywordsInput = MJBS.createInput();
const SizeInput = MJBS.createInput(); // readonly
const LikeInput = MJBS.createInput("number");
const CTimeInput = MJBS.createInput();
const UTimeInput = MJBS.createInput();
const CheckedInput = MJBS.createInput(); // readonly
const DamagedInput = MJBS.createInput(); // readonly
const DeletedInput = MJBS.createInput(); // readonly

const EditFileForm = cc("form", {
  attr: { autocomplete: "off" },
  children: [
    MJBS.createFormControl(IdInput, "ID"),
    MJBS.createFormControl(BucketInput, "Bucket ID"),
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
    MJBS.createFormControl(UTimeInput, "UTime", "更新時間, 一般填寫 Now 即可"),
    MJBS.createFormControl(
      CheckedInput,
      "Checked",
      "上次檢查檔案完整性的時間."
    ),
    MJBS.createFormControl(DamagedInput, "Damaged", "檔案是否損壞"),
    MJBS.createFormControl(
      DeletedInput,
      "Deleted",
      "檔案是否標記為刪除, 請填寫 true 或 false"
    ),
  ],
});

$("#root")
  .css(RootCss)
  .append(
    navBar.addClass("my-3"),
    m(PageAlert).addClass("my-5"),
    m(PageLoading).addClass("my-5"),
    m(EditFileForm).addClass("my-5").hide()
  );

init();

function init() {
  initEditFileForm();
}

function initEditFileForm() {
  axiosPost({
    url: "/api/file-info",
    alert: PageAlert,
    body: { id: parseInt(fileID) },
    onSuccess: (resp) => {
      const file = resp.data;

      IdInput.setVal(file.id);
      BucketInput.setVal(file.bucketid);
      NameInput.setVal(file.name);
      NotesInput.setVal(file.notes);
      KeywordsInput.setVal(file.keywords);
      SizeInput.setVal(fileSizeToString(file.size));
      LikeInput.setVal(file.like);
      CTimeInput.setVal(file.ctime);
      UTimeInput.setVal("Now");
      CheckedInput.setVal(file.checked);
      DamagedInput.setVal(file.damaged);
      DeletedInput.setVal(file.deleted);

      MJBS.disable(IdInput);
      MJBS.disable(BucketInput);
      MJBS.disable(SizeInput);
      MJBS.disable(CheckedInput);
      MJBS.disable(DamagedInput);

      EditFileForm.show();
      MJBS.focus(NotesInput);
    },
    onAlways: () => {
      PageLoading.hide();
    },
  });
}
