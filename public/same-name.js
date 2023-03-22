const SameNameRadioName = "SameNameRadio";

const OverwriteRadio = cc("input", {
  classes: "form-check-input",
  type: "radio",
  attr: { type: "radio", name: SameNameRadioName, value: "overwrite" },
});

const OverwriteRadioArea = cc("div", {
  classes: "form-check",
  children: [
    m(OverwriteRadio),
    m("label")
      .addClass("form-check-label")
      .attr({ for: OverwriteRadio.raw_id })
      .text("Overwrite (覆蓋倉庫中的檔案)"),
  ],
});

const RenameRadio = cc("input", {
  classes: "form-check-input",
  type: "radio",
  attr: { type: "radio", name: SameNameRadioName, value: "rename" },
});

const RenameRadioArea = cc("div", {
  classes: "form-check",
  children: [
    m(RenameRadio),
    m("label")
      .addClass("form-check-label")
      .attr({ for: RenameRadio.raw_id })
      .text("Rename (更改待上傳檔案的名稱)"),
  ],
});

const SameNameFile = cc('dl', {classes: 'row'});

SameNameFile.init = (file) => {
  SameNameFile.elem().append(
    m('dt').addClass('col-sm-2').text("Bucket: "),
    m('dt').addClass('col-sm-10 text-muted').text(file.bucketid),

    m('dt').addClass('col-sm-2').text("File Name: "),
    m('dt').addClass('col-sm-10 text-muted').text(file.name)
  );
  if (file.notes) {
    SameNameFile.elem().append(
      m('dt').addClass('col-sm-2').text("Notes: "),
      m('dt').addClass('col-sm-10 text-muted').text(file.notes)
    );
  }
  if (file.keywords) {
    SameNameFile.elem().append(
      m('dt').addClass('col-sm-2').text("Keywords: "),
      m('dt').addClass('col-sm-10 text-muted').text(file.keywords)
    );
  }
  SameNameFile.elem().append(
    m('dt').addClass('col-sm-2').text("Size: "),
    m('dt').addClass('col-sm-10 text-muted').text(fileSizeToString(file.size))
  );
};

const RenameInput = MJBS.createInput();
const RenameBtn = MJBS.createButton('Rename');
const RenameAlert = MJBS.createAlert();
const RenameForm = cc('div', {
  classes: 'input-group',
  children: [
    m(RenameInput),
    m(RenameBtn).on('click', (event) => {
      event.preventDefault();
      const new_name = MJBS.valOf(RenameInput, 'trim');
      if (new_name == RenameInput.old_name) {
        RenameAlert.insert('warning', '檔案名稱未變更.');
        return;
      }
      axiosPost({
        url: "/api/rename-waiting-file",
        alert: RenameAlert,
        body: {
          old_name: RenameInput.old_name,
          new_name: new_name,
        },
        onSuccess: () => {
          RenameAlert.insert('success', 'Rename Success! 三秒後自動刷新.');
          setTimeout(() => {
            window.location.reload();
          }, 3000);
        }
      });
    })
  ]
});
const RenameArea = cc("div", {
  children: [
    m('div').text('在此更改待上傳檔案的名稱, 注意保留副檔名(擴展名)').addClass('mb-1'),
    m(RenameForm).addClass('mb-2'),
    m(RenameAlert)
  ]
})

const OverwriteBtn = MJBS.createButton('Overwrite');
const OverwriteBtnArea = cc("div", {
  children: [
    span('點擊此按鈕執行覆蓋: '),
    m(OverwriteBtn),
  ]
})

const SameNameRadioCard = cc("div", {
  classes: "card",
  children: [
    m("div")
      .addClass("card-body")
      .append(
        m("div").text("待上傳檔案的名稱, 與倉庫中的檔案名稱相同:").addClass("mb-3"),
        m(SameNameFile).addClass("mb-3"),
        m("div").text("請選擇處理方式:").addClass("mb-3"),
        m(OverwriteRadioArea),
        m(RenameRadioArea),
        m(RenameArea).addClass('my-5').hide(),
        m(OverwriteBtnArea).addClass('my-5').hide(),
      ),
  ],
});

SameNameRadioCard.init = (file) => {
  $(`input[name="${SameNameRadioName}"]`).on("change", () => {
    const val = getSameNameRadioValue();
    if (val == 'rename') {
      OverwriteBtnArea.hide();
      RenameArea.show();
      MJBS.focus(RenameInput);
      return;
    }
    if (val == 'overwrite') {
      RenameArea.hide();
      OverwriteBtnArea.show();
    }
  });
  SameNameFile.init(file);
  RenameInput.elem().val(file.name);
  RenameInput.old_name = file.name;
};

function getSameNameRadioValue() {
  return $(`input[name="${SameNameRadioName}"]:checked`).val();
}
