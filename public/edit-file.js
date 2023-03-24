$("title").text("Edit File Attributes (修改檔案屬性) - Local Buckets");

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

const IdInput = MJBS.createInput('number', 'required'); // readonly
const BucketInput = MJBS.createInput();                 // readonly
const NameInput = MJBS.createInput('text', 'required');
const NotesInput = MJBS.createInput();
const KeywordsInput = MJBS.createInput();
const SizeInput = MJBS.createInput();                   // readonly
const LikeInput = MJBS.createInput('number');
const CTimeInput = MJBS.createInput();
const UTimeInput = MJBS.createInput();
const CheckedInput = MJBS.createInput();                // readonly
const DamagedInput = MJBS.createInput();                // readonly
const DeletedInput = MJBS.createInput();                // readonly

const ChangePwdForm = cc("form", {
  attr: { autocomplete: "off" },
  children: [
    MJBS.createFormControl(IdInput, "ID"),
    MJBS.createFormControl(BucketInput, "Bucket ID"),
    MJBS.createFormControl(NameInput, "File Name"),
    MJBS.createFormControl(NotesInput, "Notes", "簡單描述, 作用類似檔案名稱."),
    MJBS.createFormControl(KeywordsInput, "Keywords", "關鍵詞, 用於輔助搜尋."),
    MJBS.createFormControl(SizeInput, "Size"),
    MJBS.createFormControl(LikeInput, "Like", "點讚數, 數字越大表示該檔案越重要."),
    MJBS.createFormControl(CTimeInput, "CTime", "創建時間, 格式 2006-01-02 15:04:05Z07:00"),
    MJBS.createFormControl(UTimeInput, "UTime", "創建時間, 填寫 Now 可自動更新時間"),
    MJBS.createFormControl(CheckedInput, "Checked", "上次檢查檔案完整性的時間."),
    MJBS.createFormControl(DamagedInput, "Damaged", "檔案是否損壞"),
    MJBS.createFormControl(DeletedInput, "Deleted", "檔案是否標記為刪除"),
  ]}
);
