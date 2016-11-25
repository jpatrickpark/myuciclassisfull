$( function() {
    FULL = 0;
    OPEN = 1;
    WAITLIST = 2;
    NONEXISTENT = 3;
    DELETED = 4;
    ENTRYEXISTS = 5;
    NOTDELETED = 6;


    function createServerResponseElement(stat,code) {
        // Displays server side information about the course user just requested.
        icon = "";
        message = "";
        switch (stat) {
        case FULL:
            icon = '<i class="fa fa-check fa-5x" style="color: green;"></i>';
            message = '<h4>Course '+code+' is full! You will get notified.</h4>';
            break;
        case WAITLIST:
            icon = '<i class="fa fa-exclamation-triangle fa-5x" style="color: yellow;"></i>';
            message = '<h4>Course '+code+' has an open waitlist!</h4><h4> You can go ahead and enroll if your window is open.</h4><h4>You will still get notified in case you cannot enroll before it gets full.</h4>';
            break;
        case OPEN:
            icon = '<i class="fa fa-exclamation-triangle fa-5x" style="color: yellow;"></i>';
            message = '<h4>Course '+code+' is currently open! </h4><h4> You can go ahead and enroll if your window is open.</h4><h4>You will still get notified in case you cannot enroll before it gets full.</h4>';
            break;
        case NONEXISTENT:
            icon = '<i class="fa fa-ban fa-5x" style="color: red;"></i>'
            message = '<h4>Course '+code+' does not exist!</h4>'
            break;
        case DELETED:
            icon = '<i class="fa fa-check fa-5x" style="color: green;"></i>';
            message = '<h4>You will no longer receive notification for course '+code+'!</h4>'
            break;
        case ENTRYEXISTS:
            icon = '<i class="fa fa-ban fa-5x" style="color: red;"></i>'
            message = '<h4>'+code+' is already registered as your course!</h4>'
            break;
        case NOTDELETED:
            icon = '<i class="fa fa-ban fa-5x" style="color: red;"></i>'
            message = '<h4>'+code+' is not one of your registered courses!</h4>'
            break;
        default:
            break;
        }

        return $('<div id="serverResponse">'+icon+message+'</div>');
    };
    function createCourseListElement(courses) {
        // Displays server side information about the list of user's requested courses
        element = '<tbody id="tableBody">';
        $.each(courses, function(key,value) {
                switch (value.courseStatus) {
                case FULL:       
                    subelement = "<tr class='danger'><td>";
                    subelement = subelement.concat(value.courseCode);
                    subelement = subelement.concat("</td><td>Full");
                    break;
                case WAITLIST:
                    subelement = "<tr class='warning'><td>";
                    subelement = subelement.concat(value.courseCode);
                    subelement = subelement.concat("</td><td>Waitlist");
                    break;
                case OPEN:
                    subelement = "<tr class='success'><td>";
                    subelement = subelement.concat(value.courseCode);
                    subelement = subelement.concat("</td><td>Open");
                    break;
                default:
                    subelement = "<tr class='danger'><td>";
                    subelement = subelement.concat(value.courseCode);
                    subelement = subelement.concat("</td><td>error");
                    break;
                }
                subelement = subelement.concat("</td>");
                subelement = subelement.concat("<td><a class='deleteButton' url='/my-uci-class-is-full/term/");
                subelement = subelement.concat(value.quarter + "/" + value.courseCode);
                subelement = subelement.concat("'><i class='fa fa-trash fa-lg'></i></a></td></tr>");
            element = element.concat(subelement);
        });
        element = element.concat("</tbody>");

        return $(element);
    };
    loading_icon = $("#loading");
    loading_icon.hide();
    $(document).ajaxStart(function(){
        loading_icon.show();
    }).ajaxComplete(function(){
        loading_icon.hide();
    });
    $courseCodeForm = $("#courseCodeForm");
    $courseCode = $courseCodeForm.find('input[name="courseCode"]');
    $displayResponse = $('#status');
    $table = $('#table');

    // When user submits request, put it to the user's request set for the according term.
    $courseCodeForm.submit(function(event) {
        event.preventDefault();
        $('#serverResponse').remove();
        $.ajax({
            url: $courseCodeForm.attr('action'),
            type: 'PUT',
            data: {courseCode: $courseCode.val()},
            success: function (result) {
              $displayResponse.append(createServerResponseElement(result.status, $courseCode.val()));
              if (result.status != NONEXISTENT) {
                $('#tableBody').remove();
                $table.append(createCourseListElement(result.courses));
                addListenerToDeleteButtons();
              }
              $courseCode.val('');
            },
            error: function (textStatus, errThrown) {
                $.each(textStatus, function(key,value) {
                  if (key=='responseText') {
                    alert(value);
                  }
                });
            }
        });
    });
    function addListenerToDeleteButtons() {

    // When Delete Button is Clicked, remove according course from user's request set.
    $('.deleteButton').click(function(){
        $('#serverResponse').remove();
        url = $(this).attr('url');
        $.ajax({
            url: url,
            type: 'DELETE',
            success: function (result) {
              $displayResponse.append(createServerResponseElement(result.status, url.substr(url.length-5)));
              $('#tableBody').remove();
              $table.append(createCourseListElement(result.courses));
              addListenerToDeleteButtons();
              $courseCode.val('');
            },
            error: function (textStatus, errThrown) {
                $.each(textStatus, function(key,value) {
                  if (key=='responseText') {
                    alert(value);
                  }
                });
            }
        });
    });
    };
    addListenerToDeleteButtons();
});
